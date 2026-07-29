"""
SHAP + XGBoost feature importance analysis.

Reads from feature_store table (populated by feature_pipeline.py),
trains XGBoost to predict target (1 = +30% before -10%, 0 = -10% first),
then uses SHAP to show which features matter most.

Usage:
    python ml/feature_pipeline.py   # run first to populate feature_store
    python ml/shap_analysis.py
"""

import os
import numpy as np
import pandas as pd
import xgboost as xgb
import shap
import matplotlib.pyplot as plt
from sklearn.metrics import classification_report
from sqlalchemy import create_engine
from dotenv import load_dotenv
load_dotenv()

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/ninja_trader")
engine = create_engine(DATABASE_URL)

# 1. Load data from feature_store
print("Loading feature_store...")
df = pd.read_sql("""
    SELECT * FROM feature_store
    WHERE target IS NOT NULL
""", engine, parse_dates=['date'])
print(f"Loaded {len(df)} rows")

# 2. Define features
features = [
    'slope_20d', 'slope_60d', 'curvature_20d',
    'drawdown_magnitude', 'drawdown_duration', 'price_vs_200d_ma', 'price_range_position',
    'range_1m', 'range_3m', 'range_6m',
    'vpt_ratio_20d', 'vpt_ratio_60d', 'vol_ratio',
    'delivery_pct', 'avg_delivery_pct_20d',
    'divergence_20d', 'divergence_60d', 'divergence_zscore', 'vpt_score',
    'short_activity',
    'market_cap_log', 'industry_encoded', 'volatility_20d',
    'vol_ratio_rank'
]

# Drop rows with NaN in features — fill NaN and inf with 0
df[features] = df[features].replace([np.inf, -np.inf], np.nan).fillna(0)
print(f"After cleanup: {len(df)} rows, target=1 rate: {df['target'].mean():.3f}")

# 3. Train/test split by time
split_date = df['date'].quantile(0.75)  # use 75% of data for training
train = df[df['date'] < split_date]
test = df[df['date'] >= split_date]
print(f"Train: {len(train)} rows, Test: {len(test)} rows")

X_train, y_train = train[features], train['target']
X_test, y_test = test[features], test['target']

# 4. Train XGBoost
print("Training XGBoost...")
model = xgb.XGBClassifier(
    n_estimators=500,
    max_depth=6,
    learning_rate=0.05,
    min_child_weight=50,
    subsample=0.8,
    colsample_bytree=0.8,
    eval_metric='logloss',
    use_label_encoder=False
)
model.fit(X_train, y_train)

train_acc = model.score(X_train, y_train)
test_acc = model.score(X_test, y_test)
print(f"Train accuracy: {train_acc:.3f}")
print(f"Test accuracy:  {test_acc:.3f}")

# 5. Classification report
print("\nClassification Report (test):")
print(classification_report(y_test, model.predict(X_test)))

# 6. Feature importance (built-in)
print("\nFeature Importance (gain):")
importance = model.get_booster().get_score(importance_type='gain')
for feat in sorted(importance, key=importance.get, reverse=True):
    print(f"  {feat}: {importance[feat]:.1f}")

# 7. SHAP analysis
print("\nComputing SHAP values...")
explainer = shap.TreeExplainer(model)
shap_values = explainer.shap_values(X_test)

# Summary plot
print("Generating SHAP summary plot...")
shap.summary_plot(shap_values, X_test, show=False)
plt.tight_layout()
plt.savefig(os.path.join(SCRIPT_DIR, "shap_summary.png"), dpi=150, bbox_inches='tight')
plt.close()
print(f"Saved: {os.path.join(SCRIPT_DIR, 'shap_summary.png')}")

# Bar plot
shap.plots.bar(explainer(X_test), show=False)
plt.tight_layout()
plt.savefig(os.path.join(SCRIPT_DIR, "shap_bar.png"), dpi=150, bbox_inches='tight')
plt.close()
print(f"Saved: {os.path.join(SCRIPT_DIR, 'shap_bar.png')}")

# 8. Print top predictions on test set
print("\nTop 20 highest-probability predictions on test set:")
probs = model.predict_proba(X_test)[:, 1]
test_with_probs = test.copy()
test_with_probs['prob'] = probs
top = test_with_probs.nlargest(20, 'prob')[['symbol', 'date', 'prob', 'target'] + features[:5]]
print(top.to_string(index=False))
