import { NextRequest, NextResponse } from "next/server";
import { createDevJwt } from "../../../lib/dev-jwt";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

async function proxy(request: NextRequest, context: RouteContext) {
  try {
    const { path } = await context.params;
    const query = request.nextUrl.search || "";
    const targetPath = `/api/v1/${path.join("/")}${query}`;
    const method = request.method;
    const hasBody = !["GET", "HEAD"].includes(method);
    const sessionToken = request.cookies.get("ispboss_access_token")?.value;
    const response = await fetch(`${billingApiBaseUrl}${targetPath}`, {
      method,
      headers: {
        "Content-Type": request.headers.get("content-type") || "application/json",
        Authorization: sessionToken ? `Bearer ${sessionToken}` : `Bearer ${createDevJwt()}`,
      },
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });

    const contentType = response.headers.get("content-type") || "";
    if (contentType.includes("application/json")) {
      const text = await response.text();
      const body = text ? JSON.parse(text) : null;
      return NextResponse.json(body, { status: response.status });
    }

    const body = await response.arrayBuffer();
    return new NextResponse(body, {
      status: response.status,
      headers: {
        "Content-Type": contentType || "application/octet-stream",
        ...(response.headers.get("content-disposition")
          ? { "Content-Disposition": response.headers.get("content-disposition") as string }
          : {}),
      },
    });
  } catch (error) {
    return NextResponse.json(
      {
        success: false,
        error: {
          code: "BILLING_API_ERROR",
          message: error instanceof Error ? error.message : "Billing API request failed",
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
