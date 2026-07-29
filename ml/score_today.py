"""
Score all stocks using the latest available data and rank by probability
of hitting +30% before -10%.

Usage:
    poetry run python ml/score_today.py
"""

import os
import numpy as np
import pandas as pd
import xgboost as xgb
from sqlalchemy import create_engine
from dotenv import load_dotenv
load_dotenv()

DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://localhost:5432/ninja_trader")
engine = create_engine(DATABASE_URL)

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

# 1. Load training data (rows where target is known)
print("Loading training data...")
train_df = pd.read_sql("SELECT * FROM feature_store WHERE target IS NOT NULL", engine, parse_dates=['date'])
train_df[features] = train_df[features].replace([np.inf, -np.inf], np.nan).fillna(0)
print(f"  Training rows: {len(train_df)}, target=1 rate: {train_df['target'].mean():.3f}")

# 2. Train model
print("Training model...")
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
model.fit(train_df[features], train_df['target'])

# 3. Load latest data (most recent date per symbol)
print("Loading latest data...")
latest = pd.read_sql("""
    SELECT * FROM feature_store
    WHERE date = (SELECT MAX(date) FROM feature_store)
""", engine, parse_dates=['date'])
latest[features] = latest[features].replace([np.inf, -np.inf], np.nan).fillna(0)

# 4. Score
probs = model.predict_proba(latest[features])[:, 1]
latest['probability'] = probs

# 5. Display top picks
top = latest.nlargest(30, 'probability')[[
    'symbol', 'date', 'probability',
    'vpt_score', 'divergence_20d', 'vol_ratio', 'delivery_pct',
    'drawdown_magnitude', 'price_vs_200d_ma', 'vpt_ratio_60d'
]]

print(f"\nDate: {latest['date'].iloc[0].strftime('%Y-%m-%d')}")
print(f"Total stocks scored: {len(latest)}")
print(f"\nTop 30 stocks most likely to hit +30% before -10%:\n")
print(top.to_string(index=False))
