export function AppVersion() {
  const version = import.meta.env.VITE_APP_VERSION || 'development';

  return (
    <footer className="border-t border-zinc-200 px-6 py-3 text-right text-xs text-zinc-500">
      Aegis {version}
    </footer>
  );
}
