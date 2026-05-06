import { ImageResponse } from "next/og";

export const size = {
  width: 180,
  height: 180,
};

export const contentType = "image/png";

export default function AppleIcon() {
  return new ImageResponse(
    (
      <div
        style={{
          width: "100%",
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          borderRadius: 40,
          background: "linear-gradient(135deg, #2563eb 0%, #0f172a 100%)",
          color: "white",
          fontFamily: "Arial, sans-serif",
          fontSize: 68,
          fontWeight: 800,
        }}
      >
        IB
      </div>
    ),
    size,
  );
}
