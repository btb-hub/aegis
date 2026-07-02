import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { formatDuration, type MetricBucket } from '../../lib/analyticsTypes';
import { formatShortDate } from '../../lib/formatDate';

type MetricTrendChartProps = {
  series: MetricBucket[];
  ariaLabel: string;
  emptyLabel: string;
};

export function MetricTrendChart({ series, ariaLabel, emptyLabel }: MetricTrendChartProps) {
  if (series.length === 0) {
    return <p className="text-sm text-zinc-600">{emptyLabel}</p>;
  }

  const data = series.map((bucket) => ({
    label: formatShortDate(bucket.bucket_start),
    meanSeconds: bucket.mean_seconds,
  }));

  return (
    <div className="h-48 w-full" role="img" aria-label={ariaLabel}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e4e4e7" />
          <XAxis dataKey="label" tick={{ fontSize: 12 }} stroke="#71717a" />
          <YAxis
            tick={{ fontSize: 12 }}
            stroke="#71717a"
            tickFormatter={(value: number) => formatDuration(value)}
          />
          <Tooltip
            formatter={(value) => formatDuration(typeof value === 'number' ? value : Number(value ?? 0))}
          />
          <Line type="monotone" dataKey="meanSeconds" stroke="#2563eb" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
