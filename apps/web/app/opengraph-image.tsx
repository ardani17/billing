import { ImageResponse } from "next/og";

export const size = {
  width: 1200,
  height: 630,
};

export const contentType = "image/png";

export default function OpenGraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          background: "#07111f",
          color: "white",
          fontFamily: "Arial, sans-serif",
          position: "relative",
          overflow: "hidden",
        }}
      >
        <div
          style={{
            position: "absolute",
            inset: 0,
            background:
              "radial-gradient(circle at 78% 26%, rgba(37, 99, 235, 0.42), transparent 34%), linear-gradient(135deg, #0f172a 0%, #111827 54%, #020617 100%)",
          }}
        />
        <div
          style={{
            position: "absolute",
            right: 70,
            top: 90,
            width: 430,
            height: 370,
            border: "1px solid rgba(255,255,255,0.18)",
            borderRadius: 28,
            background: "rgba(15, 23, 42, 0.82)",
            boxShadow: "0 45px 110px rgba(0,0,0,0.34)",
            padding: 28,
            display: "flex",
            flexDirection: "column",
            gap: 24,
          }}
        >
          <div style={{ display: "flex", gap: 12 }}>
            <div style={{ width: 12, height: 12, borderRadius: 99, background: "#f87171" }} />
            <div style={{ width: 12, height: 12, borderRadius: 99, background: "#fbbf24" }} />
            <div style={{ width: 12, height: 12, borderRadius: 99, background: "#34d399" }} />
          </div>
          <div style={{ display: "flex", gap: 16 }}>
            {["847", "48,7jt", "31"].map((value) => (
              <div
                key={value}
                style={{
                  flex: 1,
                  borderTop: "1px solid rgba(148,163,184,0.28)",
                  paddingTop: 18,
                  fontSize: 36,
                  fontWeight: 700,
                }}
              >
                {value}
              </div>
            ))}
          </div>
          <div style={{ display: "flex", alignItems: "flex-end", gap: 12, height: 150 }}>
            {[42, 64, 55, 79, 71, 92, 84].map((bar, index) => (
              <div
                key={index}
                style={{
                  flex: 1,
                  height: `${bar}%`,
                  background: "#3b82f6",
                  borderRadius: "10px 10px 0 0",
                }}
              />
            ))}
          </div>
          <div
            style={{
              height: 76,
              borderRadius: 18,
              border: "1px solid rgba(148,163,184,0.22)",
              background:
                "repeating-linear-gradient(90deg, transparent, transparent 23px, rgba(148,163,184,0.18) 24px), repeating-linear-gradient(0deg, transparent, transparent 23px, rgba(148,163,184,0.18) 24px)",
            }}
          />
        </div>
        <div
          style={{
            position: "relative",
            zIndex: 1,
            padding: "74px 76px",
            width: 720,
            display: "flex",
            flexDirection: "column",
            justifyContent: "space-between",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 18 }}>
            <div
              style={{
                width: 64,
                height: 64,
                borderRadius: 18,
                background: "#2563eb",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontSize: 24,
                fontWeight: 800,
              }}
            >
              IB
            </div>
            <div style={{ fontSize: 34, fontWeight: 750 }}>ISPBoss</div>
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 26 }}>
            <div style={{ color: "#93c5fd", fontSize: 22, fontWeight: 700 }}>
              Platform billing ISP Indonesia
            </div>
            <div style={{ fontSize: 72, lineHeight: 0.96, fontWeight: 780, letterSpacing: -2 }}>
              Billing dan jaringan ISP dalam satu dashboard
            </div>
            <div style={{ color: "#cbd5e1", fontSize: 27, lineHeight: 1.35 }}>
              Pelanggan, invoice, pembayaran, MikroTik, OLT, notifikasi, dan peta FTTH.
            </div>
          </div>
        </div>
      </div>
    ),
    size,
  );
}
