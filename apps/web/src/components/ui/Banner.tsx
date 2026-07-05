type BannerVariant = 'info' | 'warning' | 'error';

type BannerProps = {
  variant?: BannerVariant;
  children: string;
};

const variantClass: Record<BannerVariant, string> = {
  info: 'border-zinc-200 bg-zinc-50 text-zinc-800',
  warning: 'border-amber-200 bg-amber-50 text-amber-950',
  error: 'border-red-200 bg-red-50 text-red-900',
};

export function Banner({ variant = 'info', children }: BannerProps) {
  return (
    <div role="alert" className={`rounded-md border px-4 py-3 text-sm ${variantClass[variant]}`}>
      {children}
    </div>
  );
}
