import type { Meta, StoryObj } from '@storybook/react';
import { MetricTrendChart } from './MetricTrendChart';

const meta: Meta<typeof MetricTrendChart> = {
  title: 'Analytics/MetricTrendChart',
  component: MetricTrendChart,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof MetricTrendChart>;

export const WithData: Story = {
  args: {
    ariaLabel: 'Mean time to acknowledge trend',
    emptyLabel: 'No data',
    series: [
      { bucket_start: '2026-06-01T00:00:00Z', mean_seconds: 120, count: 2 },
      { bucket_start: '2026-06-02T00:00:00Z', mean_seconds: 90, count: 3 },
      { bucket_start: '2026-06-03T00:00:00Z', mean_seconds: 60, count: 1 },
    ],
  },
};

export const Empty: Story = {
  args: {
    ariaLabel: 'Mean time to acknowledge trend',
    emptyLabel: 'No data in this range',
    series: [],
  },
};
