import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MetricTrendChart } from './MetricTrendChart';

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  LineChart: ({ children }: { children: React.ReactNode }) => <div data-testid="line-chart">{children}</div>,
  CartesianGrid: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  Line: () => null,
}));

describe('MetricTrendChart', () => {
  it('renders empty state', () => {
    render(<MetricTrendChart series={[]} ariaLabel="MTTA" emptyLabel="No data" />);
    expect(screen.getByText('No data')).toBeInTheDocument();
  });

  it('renders chart when series has buckets', () => {
    render(
      <MetricTrendChart
        series={[{ bucket_start: '2026-06-01T00:00:00Z', mean_seconds: 120, count: 2 }]}
        ariaLabel="MTTA"
        emptyLabel="No data"
      />,
    );
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });
});
