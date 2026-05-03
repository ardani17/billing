import { NextRequest, NextResponse } from "next/server";
import { createDevJwt } from "../../../lib/dev-jwt";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

const networkApiBaseUrl =
  process.env.NETWORK_API_URL || "http://localhost:3002";

async function proxy(request: NextRequest, context: RouteContext) {
  try {
    const { path } = await context.params;
    const query = request.nextUrl.search || "";
    const targetPath = `/api/v1/network-map/${path.join("/")}${query}`;
    const method = request.method;
    const hasBody = !["GET", "HEAD"].includes(method);
    const body = hasBody ? await request.arrayBuffer() : undefined;
    const headers = new Headers();

    headers.set("Authorization", `Bearer ${createDevJwt()}`);
    const contentType = request.headers.get("content-type");
    if (contentType) {
      headers.set("Content-Type", contentType);
    }

    const response = await fetch(`${networkApiBaseUrl}${targetPath}`, {
      method,
      headers,
      body,
      cache: "no-store",
    });

    if (response.status === 204) {
      return new NextResponse(null, { status: 204 });
    }

    const responseContentType = response.headers.get("content-type") || "";
    if (responseContentType.includes("application/json")) {
      const data = await response.json();
      return NextResponse.json(data, { status: response.status });
    }

    const bytes = await response.arrayBuffer();
    return new NextResponse(bytes, {
      status: response.status,
      headers: {
        "Content-Type": responseContentType || "application/octet-stream",
      },
    });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "NETWORK_MAP_API_ERROR",
          message:
            error instanceof Error
              ? error.message
              : "Network map API request failed",
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
