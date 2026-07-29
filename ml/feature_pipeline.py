"""
Feature pipeline: reads historicaldata from PostgreSQL, computes features,
and writes to a feature_store table for ML training.

Target: 1 if price reaches +30% before dropping -10%, 0 otherwise.

Usage:
    pip install pandas numpy sqlalchemy psycopg2-binary scikit-learn
    python ml/feature_pipeline.py
"""
from dotenv import load_dotenv


import os
import numpy as np
import pandas as pd
from numba import njit
from sqlalchemy import create_engine
from sklearn.preprocessing import LabelEncoder

load_dotenv()
DATABASE_URL = os.getenv("DATABASE_URL")
print(DATABASE_URL)

engine = create_engine(DATABASE_URL)


def load_data():
    """Load all required data from PostgreSQL."""
    print("Loading historicaldata...")
    hist = pd.read_sql("""
        SELECT symbol, date, o, h, l, c, adjusted_close_price, volume,
               no_of_trades, volume_per_trade, delivery_percentage,
               year_high, year_low, vpt_score, vpt_ma20, volume_ma20, divergence
        FROM historicaldata
        WHERE adjusted_close_price > 0
        ORDER BY symbol, date
    """, engine, parse_dates=['date'])
    print(f"  {len(hist)} rows")

    print("Loading shorts...")
    shorts = pd.read_sql("""
        SELECT trading_symbol AS symbol, date, quantity
        FROM shorts
        ORDER BY trading_symbol, date
    """, engine, parse_dates=['date'])
    print(f"  {len(shorts)} rows")

    print("Loading instruments...")
    instruments = pd.read_sql("""
        SELECT trading_symbol AS symbol, market_cap, basic_industry
        FROM instruments
        WHERE active = true
    """, engine)
    print(f"  {len(instruments)} rows")

    return hist, shorts, instruments


@njit
def compute_target_numba(prices):
    """
    For each day, look forward:
    - If price reaches +30% before dropping -10% → 1
    - If price drops -10% before reaching +30% → 0
    - If neither hit yet → -1 (will be converted to NaN)
    """
    n = len(prices)
    targets = np.full(n, -1.0)

    for i in range(n):
        entry_price = prices[i]
        upper = entry_price * 1.30
        lower = entry_price * 0.90

        for j in range(i + 1, n):
            if prices[j] >= upper:
                targets[i] = 1.0
                break
            if prices[j] <= lower:
                targets[i] = 0.0
                break

    return targets


def compute_slope_vectorized(prices, window):
    """Compute linear regression slope using vectorized numpy operations."""
    n = len(prices)
    slopes = np.full(n, np.nan)
    t = np.arange(window, dtype=float)
    t_mean = t.mean()
    t_var = ((t - t_mean) ** 2).sum()

    for i in range(window - 1, n):
        y = prices[i - window + 1:i + 1]
        if np.isnan(y).any():
            continue
        slopes[i] = ((t - t_mean) * (y - y.mean())).sum() / t_var

    return slopes


def compute_features_for_symbol(df):
    """Compute all features for a single symbol's dataframe."""
    price = df['adjusted_close_price'].values
    vpt = df['volume_per_trade'].values.astype(float)
    volume = df['volume'].values.astype(float)
    delivery = df['delivery_percentage'].values.astype(float)
    n = len(price)

    features = pd.DataFrame(index=df.index)

    # Price features
    slope_20 = compute_slope_vectorized(price, 20)
    slope_60 = compute_slope_vectorized(price, 60)
    features['slope_20d'] = slope_20
    features['slope_60d'] = slope_60

    # Curvature: change in slope over 5 days
    curvature = np.full(n, np.nan)
    for i in range(5, n):
        if not np.isnan(slope_20[i]) and not np.isnan(slope_20[i - 5]):
            curvature[i] = slope_20[i] - slope_20[i - 5]
    features['curvature_20d'] = curvature

    # Drawdown
    rolling_max = np.maximum.accumulate(price)
    features['drawdown_magnitude'] = (price - rolling_max) / rolling_max

    # Drawdown duration: count consecutive days below rolling max
    duration = np.zeros(n)
    for i in range(1, n):
        if price[i] < rolling_max[i]:
            duration[i] = duration[i - 1] + 1
    features['drawdown_duration'] = duration

    # Price vs 200d MA
    ma200 = pd.Series(price).rolling(200).mean().values
    features['price_vs_200d_ma'] = price / ma200

    # Price range position (using year_high/year_low from DB)
    year_high = df['year_high'].values
    year_low = df['year_low'].values
    range_val = year_high - year_low
    range_val[range_val == 0] = np.nan
    features['price_range_position'] = (price - year_low) / range_val

    # Price range % (max - min) / min over rolling windows
    price_series = pd.Series(price)
    for window, name in [(20, 'range_1m'), (60, 'range_3m'), (120, 'range_6m')]:
        rolling_max = price_series.rolling(window).max()
        rolling_min = price_series.rolling(window).min()
        features[name] = ((rolling_max - rolling_min) / rolling_min.replace(0, np.nan) * 100).values

    # Volume features
    vpt_ma20 = pd.Series(vpt).rolling(20).mean().values
    vpt_ma60 = pd.Series(vpt).rolling(60).mean().values
    features['vpt_ratio_20d'] = vpt / np.where(vpt_ma20 > 0, vpt_ma20, np.nan)
    features['vpt_ratio_60d'] = vpt / np.where(vpt_ma60 > 0, vpt_ma60, np.nan)

    vol_ma20 = pd.Series(volume).rolling(20).mean().values
    features['vol_ratio'] = volume / np.where(vol_ma20 > 0, vol_ma20, np.nan)

    features['delivery_pct'] = delivery
    features['avg_delivery_pct_20d'] = pd.Series(delivery).rolling(20).mean().values

    # Divergence: VPT % change vs price % change
    vpt_ma20_series = pd.Series(vpt).rolling(20).mean()
    features['divergence_20d'] = (vpt_ma20_series.pct_change(20) - pd.Series(price).pct_change(20)).values

    vpt_ma60_series = pd.Series(vpt).rolling(60).mean()
    features['divergence_60d'] = (vpt_ma60_series.pct_change(60) - pd.Series(price).pct_change(60)).values

    features['vpt_score'] = df['vpt_score'].values
    features['divergence_zscore'] = df.get('divergence', pd.Series(np.zeros(n), index=df.index)).values

    # Volatility
    prev_price = np.roll(price, 1)
    prev_price[0] = np.nan
    daily_returns = (price - prev_price) / np.where(prev_price > 0, prev_price, np.nan)
    features['volatility_20d'] = pd.Series(daily_returns).rolling(20).std().values

    return features


def run_pipeline():
    hist, shorts, instruments = load_data()

    # Encode industry
    le = LabelEncoder()
    instruments['industry_encoded'] = le.fit_transform(instruments['basic_industry'].fillna('Unknown'))
    industry_map = instruments.set_index('symbol')[['market_cap', 'industry_encoded']]

    # Merge shorts into historicaldata
    shorts_daily = shorts.groupby(['symbol', 'date'])['quantity'].sum().reset_index()
    hist = hist.merge(shorts_daily, on=['symbol', 'date'], how='left')
    hist['quantity'] = hist['quantity'].fillna(0)

    print("Computing features per symbol...")
    all_features = []
    symbols = hist['symbol'].unique()

    for i, symbol in enumerate(symbols):
        if (i + 1) % 100 == 0:
            print(f"  Processing {i+1}/{len(symbols)}...")

        sym_df = hist[hist['symbol'] == symbol].copy().reset_index(drop=True)
        if len(sym_df) < 60:
            continue

        # Compute features
        features = compute_features_for_symbol(sym_df)

        # Short activity
        short_qty = sym_df['quantity'].values.astype(float)
        short_ma60 = pd.Series(short_qty).rolling(60).mean().values
        features['short_activity'] = short_qty / np.where(short_ma60 > 0, short_ma60, np.nan)

        # Market context (static per symbol)
        if symbol in industry_map.index:
            info = industry_map.loc[symbol]
            features['market_cap_log'] = np.log1p(info['market_cap'])
            features['industry_encoded'] = info['industry_encoded']
        else:
            features['market_cap_log'] = np.nan
            features['industry_encoded'] = -1

        # Target
        targets = compute_target_numba(sym_df['adjusted_close_price'].values)
        targets[targets == -1] = np.nan
        features['target'] = targets

        # Metadata
        features['symbol'] = symbol
        features['date'] = sym_df['date'].values

        all_features.append(features)

    print("Combining all symbols...")
    result = pd.concat(all_features, ignore_index=True)

    # Cross-sectional feature: vol_ratio rank per day
    result['vol_ratio_rank'] = result.groupby('date')['vol_ratio'].rank(pct=True)

    # Stats
    result_with_target = result.dropna(subset=['target'])
    print(f"Total rows: {len(result)}, with target: {len(result_with_target)}")
    if len(result_with_target) > 0:
        print(f"Target distribution: 1={result_with_target['target'].sum():.0f} ({result_with_target['target'].mean()*100:.1f}%), 0={len(result_with_target) - result_with_target['target'].sum():.0f}")

    # Write to DB
    print("Writing to feature_store table...")
    result.to_sql('feature_store', engine, if_exists='replace', index=False,
                  method='multi', chunksize=5000)
    print(f"Done. Wrote {len(result)} rows to feature_store.")


if __name__ == '__main__':
    run_pipeline()
