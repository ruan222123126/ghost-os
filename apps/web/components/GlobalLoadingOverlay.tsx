const DELAYS = [0.2, 0.3, 0.4, 0.1, 0.2, 0.3, 0.0, 0.1, 0.2] as const;

export function GlobalLoadingOverlay() {
  return (
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center bg-white/80 backdrop-blur-sm dark:bg-zinc-950/80"
      role="status"
      aria-live="polite"
      aria-label="Loading"
    >
      <style>{`
        @keyframes shrinkWave {
          0%, 50%, 100% { transform: scale(1); border-radius: 2px; }
          25% { transform: scale(0.18); border-radius: 9999px; }
        }
      `}</style>
      <div className="grid h-16 w-16 grid-cols-3 grid-rows-3 gap-0.5 md:h-20 md:w-20" aria-hidden="true">
        {DELAYS.map((delay, index) => (
          <div
            key={index}
            className="h-full w-full origin-center bg-black shadow-sm dark:bg-zinc-800"
            style={{
              animation: 'shrinkWave 1.2s infinite ease-in-out',
              animationDelay: `${delay}s`,
              willChange: 'transform',
            }}
          />
        ))}
      </div>
    </div>
  );
}
