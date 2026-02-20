import { createFileRoute, notFound } from '@tanstack/react-router';
import { ImageResponse } from '@takumi-rs/image-response';
import { source } from '@/lib/source';

const LOBSTER_EMOJI_URL =
  'https://cdn.jsdelivr.net/gh/twitter/twemoji@14.0.2/assets/72x72/1f99e.png';

function OGImage({ title, description }: { title: string; description?: string }) {
  return (
    <div
      style={{
        display: 'flex',
        width: '100%',
        height: '100%',
        backgroundColor: '#09090b',
        position: 'relative',
        overflow: 'hidden',
      }}
    >
      {/* Gradient orb bottom-left */}
      <div
        style={{
          position: 'absolute',
          bottom: '-120px',
          left: '-80px',
          width: '500px',
          height: '500px',
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(200,40,40,0.4) 0%, transparent 70%)',
        }}
      />
      {/* Gradient orb top-right */}
      <div
        style={{
          position: 'absolute',
          top: '-160px',
          right: '-100px',
          width: '400px',
          height: '400px',
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(200,40,40,0.15) 0%, transparent 70%)',
        }}
      />
      {/* Subtle top border accent */}
      <div
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: '4px',
          background: 'linear-gradient(to right, transparent, rgb(220,60,60), transparent)',
        }}
      />
      {/* Content */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'space-between',
          width: '100%',
          height: '100%',
          padding: '56px 64px',
          position: 'relative',
        }}
      >
        {/* Top: site branding */}
        <div
          style={{
            display: 'flex',
            flexDirection: 'row',
            alignItems: 'center',
            gap: '14px',
          }}
        >
          <img src={LOBSTER_EMOJI_URL} width={40} height={40} />
          <p
            style={{
              fontSize: '28px',
              fontWeight: 500,
              margin: 0,
              color: 'rgba(255,255,255,0.5)',
              letterSpacing: '-0.02em',
            }}
          >
            OpenClaw Terraform Provider
          </p>
        </div>

        {/* Middle: title + description */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <p
            style={{
              fontWeight: 800,
              fontSize: title.length > 30 ? '64px' : '76px',
              margin: 0,
              color: 'white',
              letterSpacing: '-0.03em',
              lineHeight: 1.1,
            }}
          >
            {title}
          </p>
          {description && (
            <p
              style={{
                fontSize: '32px',
                fontWeight: 400,
                color: 'rgba(255,255,255,0.55)',
                margin: 0,
                lineHeight: 1.4,
                letterSpacing: '-0.01em',
              }}
            >
              {description.length > 90 ? description.slice(0, 90) + '...' : description}
            </p>
          )}
        </div>

        {/* Bottom: GitHub repo */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="rgba(255,255,255,0.3)">
            <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
          </svg>
          <p
            style={{
              fontSize: '22px',
              fontWeight: 400,
              margin: 0,
              color: 'rgba(255,255,255,0.3)',
              fontFamily: 'Geist Mono',
            }}
          >
            kylemclaren/terraform-provider-openclaw
          </p>
        </div>
      </div>
    </div>
  );
}

export const Route = createFileRoute('/og/docs/$')({
  server: {
    handlers: {
      GET: async ({ params }) => {
        const slugParts = params._splat?.split('/') ?? [];
        const pageSlug = slugParts.slice(0, -1); // strip trailing 'image.webp'
        const page = source.getPage(pageSlug);
        if (!page) throw notFound();

        return new ImageResponse(
          <OGImage title={page.data.title} description={page.data.description} />,
          { width: 1200, height: 630, format: 'webp' },
        );
      },
    },
  },
});
