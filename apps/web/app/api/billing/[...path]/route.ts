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
    const contentType = request.headers.get("content-type") || "";
    const headers = new Headers({
      Authorization: sessionToken ? `Bearer ${sessionToken}` : `Bearer ${createDevJwt()}`,
    });
    if (contentType) headers.set("Content-Type", contentType);

    const response = await fetch(`${billingApiBaseUrl}${targetPath}`, {
      method,
      headers,
      body: hasBody ? await request.arrayBuffer() : undefined,
      cache: "no-store",
    });

    const responseType = response.headers.get("content-type") || "";
    const disposition = response.headers.get("content-disposition");
    const body = await response.arrayBuffer();

    if (responseType.includes("application/json")) {
      const text = new TextDecoder().decode(body);
      return NextResponse.json(JSON.parse(text || "{}"), { status: response.status });
    }

    const outHeaders = new Headers();
    if (responseType) outHeaders.set("Content-Type", responseType);
    if (disposition) outHeaders.set("Content-Disposition", disposition);
    return new NextResponse(body, { status: response.status, headers: outHeaders });
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
