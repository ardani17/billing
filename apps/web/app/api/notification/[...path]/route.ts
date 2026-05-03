import { NextRequest, NextResponse } from "next/server";
import { createDevJwt } from "../../../lib/dev-jwt";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

const notificationApiBaseUrl =
  process.env.NOTIFICATION_API_URL || "http://localhost:3003";

async function proxy(request: NextRequest, context: RouteContext) {
  try {
    const { path } = await context.params;
    const targetPath = `/api/v1/${path.join("/")}${request.nextUrl.search || ""}`;
    const method = request.method;
    const hasBody = !["GET", "HEAD"].includes(method);
    const body = hasBody ? await request.arrayBuffer() : undefined;
    const headers = new Headers();
    const contentType = request.headers.get("content-type");

    headers.set("Authorization", `Bearer ${createDevJwt()}`);
    if (contentType) headers.set("Content-Type", contentType);

    const response = await fetch(`${notificationApiBaseUrl}${targetPath}`, {
      method,
      headers,
      body,
      cache: "no-store",
    });

    if (response.status === 204) return new NextResponse(null, { status: 204 });

    const responseType = response.headers.get("content-type") || "";
    if (responseType.includes("application/json")) {
      return NextResponse.json(await response.json(), { status: response.status });
    }

    return new NextResponse(await response.arrayBuffer(), {
      status: response.status,
      headers: { "Content-Type": responseType || "application/octet-stream" },
    });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NOTIFICATION_API_ERROR",
          message:
            error instanceof Error ? error.message : "Notification API request failed",
        },
      },
      { status: 502 },
    );
  }
}

export async function GET(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function PUT(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}

export async function DELETE(request: NextRequest, context: RouteContext) {
  return proxy(request, context);
}
