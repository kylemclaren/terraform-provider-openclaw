'use client';

function Box({
  children,
  variant = 'default',
}: {
  children: React.ReactNode;
  variant?: 'primary' | 'default' | 'muted';
}) {
  const styles = {
    primary:
      'bg-fd-primary/10 border-fd-primary/40 text-fd-primary font-semibold',
    default:
      'bg-fd-card border-fd-border text-fd-card-foreground',
    muted:
      'bg-fd-muted border-fd-border text-fd-muted-foreground',
  };

  return (
    <div
      className={`rounded-lg border px-4 py-3 text-center text-sm ${styles[variant]}`}
    >
      {children}
    </div>
  );
}

function Connector({ className = '' }: { className?: string }) {
  return (
    <div className={`flex justify-center ${className}`}>
      <div className="w-px h-6 bg-fd-border" />
    </div>
  );
}

function Arrow({ className = '' }: { className?: string }) {
  return (
    <div className={`flex justify-center ${className}`}>
      <div className="flex flex-col items-center">
        <div className="w-px h-5 bg-fd-border" />
        <div className="w-0 h-0 border-l-[5px] border-r-[5px] border-t-[5px] border-l-transparent border-r-transparent border-t-fd-border" />
      </div>
    </div>
  );
}

export function ArchitectureDiagram() {
  return (
    <div className="not-prose my-6 rounded-xl border border-fd-border bg-fd-card/50 p-6 md:p-8">
      <div className="mx-auto max-w-md flex flex-col items-stretch">
        {/* Terraform CLI */}
        <Box variant="primary">
          <div className="text-xs uppercase tracking-wider opacity-70 mb-0.5">
            Infrastructure as Code
          </div>
          Terraform CLI
        </Box>

        <Arrow />

        {/* Provider */}
        <Box variant="default">
          <code className="text-xs font-mono">openclaw</code> provider
        </Box>

        {/* Split */}
        <div className="flex justify-center">
          <div className="flex flex-col items-center">
            <div className="w-px h-4 bg-fd-border" />
            <div className="flex items-start">
              <div className="w-24 md:w-32 h-px bg-fd-border" />
              <div className="w-px h-px" />
              <div className="w-24 md:w-32 h-px bg-fd-border" />
            </div>
          </div>
        </div>

        {/* Two branches */}
        <div className="grid grid-cols-2 gap-3 md:gap-4">
          {/* WS branch */}
          <div className="flex flex-col">
            <Arrow />
            <Box variant="default">
              <div className="font-medium">WSClient</div>
              <div className="text-xs text-fd-muted-foreground mt-0.5">
                Live JSON-RPC
              </div>
            </Box>
            <Arrow />
            <Box variant="muted">
              <div className="font-medium">OpenClaw Gateway</div>
              <div className="text-xs text-fd-muted-foreground mt-0.5 font-mono">
                :18789
              </div>
            </Box>
          </div>

          {/* File branch */}
          <div className="flex flex-col">
            <Arrow />
            <Box variant="default">
              <div className="font-medium">FileClient</div>
              <div className="text-xs text-fd-muted-foreground mt-0.5">
                Direct read/write
              </div>
            </Box>
            <Arrow />
            <Box variant="muted">
              <div className="font-medium font-mono text-xs">~/.openclaw/</div>
              <div className="text-xs text-fd-muted-foreground mt-0.5 font-mono">
                openclaw.json
              </div>
            </Box>
          </div>
        </div>
      </div>
    </div>
  );
}
