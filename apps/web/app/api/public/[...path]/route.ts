import { NextRequest, NextResponse } from "next/server";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const query = request.nextUrl.search || "";
  const targetUrl = `${billingApiBaseUrl}/api/v1/public/${path.join("/")}${query}`;
  const method = request.method;
  const hasBody = !["GET", "HEAD"].includes(method);

  try {
    const response = await fetch(targetUrl, {
      method,
      headers: {
        ...(request.headers.get("content-type") ? { "Content-Type": request.headers.get("content-type") as string } : {}),
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });

    const contentType = response.headers.get("content-type") || "";
    const body = contentType.includes("application/json") ? await response.json() : await response.text();

    if (contentType.includes("application/json")) {
      return NextResponse.json(body, { status: response.status });
    }

    return new NextResponse(body, {
      status: response.status,
      headers: { "Content-Type": contentType || "text/plain; charset=utf-8" },
    });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "PUBLIC_API_ERROR",
          message: error instanceof Error ? error.message : "Public API request failed",
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
