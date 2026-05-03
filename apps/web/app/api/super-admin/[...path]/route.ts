import { NextRequest, NextResponse } from "next/server";
import { createDevJwt } from "../../../lib/dev-jwt";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const query = request.nextUrl.search || "";
  const targetPath = path.join("/");
  const method = request.method;
  const hasBody = !["GET", "HEAD"].includes(method);

  try {
    const response = await fetch(`${billingApiBaseUrl}/api/v1/admin/${targetPath}${query}`, {
      method,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${createDevJwt({ role: "super_admin" })}`,
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
          code: "SUPER_ADMIN_API_ERROR",
          message: error instanceof Error ? error.message : "Super admin API request failed",
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
