import { ImageResponse } from "next/og";

export const alt = "Chosen ERP · A escolha inteligente para a igreja";
export const size = { width: 1200, height: 630 };
export const contentType = "image/png";

export default function OpengraphImage() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: "72px",
          background:
            "linear-gradient(135deg, #082f49 0%, #0c4a6e 45%, #0369a1 100%)",
          color: "white",
          fontFamily: "sans-serif",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 18 }}>
          <div
            style={{
              width: 64,
              height: 64,
              borderRadius: 18,
              background: "linear-gradient(135deg,#38bdf8,#0284c7)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 40,
            }}
          >
            ✝
          </div>
          <div style={{ display: "flex", flexDirection: "column" }}>
            <span style={{ fontSize: 30, fontWeight: 700 }}>Chosen</span>
            <span
              style={{
                fontSize: 16,
                letterSpacing: 6,
                color: "#7dd3fc",
                fontWeight: 600,
              }}
            >
              ERP
            </span>
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 20 }}>
          <div style={{ fontSize: 64, fontWeight: 700, lineHeight: 1.1 }}>
            A escolha inteligente para a sua igreja
          </div>
          <div style={{ fontSize: 30, color: "#bae6fd", maxWidth: 900 }}>
            Secretaria, financeiro, eventos, ministérios, governança e
            comunicação em uma só plataforma.
          </div>
        </div>

        <div style={{ display: "flex", gap: 14, fontSize: 22 }}>
          {["Multi-filial", "LGPD", "WhatsApp", "Auditoria"].map((t) => (
            <div
              key={t}
              style={{
                padding: "10px 20px",
                borderRadius: 999,
                background: "rgba(255,255,255,0.12)",
                border: "1px solid rgba(255,255,255,0.2)",
              }}
            >
              {t}
            </div>
          ))}
        </div>
      </div>
    ),
    { ...size },
  );
}
