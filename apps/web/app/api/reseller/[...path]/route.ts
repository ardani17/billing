import { NextRequest, NextResponse } from "next/server";

const billingApiBaseUrl = process.env.BILLING_API_URL || "http://localhost:3001";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

function copyForwardedHeaders(request: NextRequest) {
  const headers: Record<string, string> = {};
  const authorization = request.headers.get("authorization");
  const contentType = request.headers.get("content-type");

  if (authorization) headers.Authorization = authorization;
  if (contentType) headers["Content-Type"] = contentType;

  return headers;
}

function jsonError(status: number, code: string, message: string) {
  return NextResponse.json(
    {
      success: false,
      error: { code, message },
    },
    { status },
  );
}

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const query = request.nextUrl.search || "";
  const targetUrl = `${billingApiBaseUrl}/api/v1/reseller/${path.join("/")}${query}`;
  const method = request.method;
  const hasBody = !["GET", "HEAD"].includes(method);

  try {
    const response = await fetch(targetUrl, {
      method,
      headers: copyForwardedHeaders(request),
      body: hasBody ? await request.text() : undefined,
      cache: "no-store",
    });

    const contentType = response.headers.get("content-type") || "";

    if (contentType.includes("application/json")) {
      const text = await response.text();
      try {
        const body = text ? JSON.parse(text) : null;
        return NextResponse.json(body, { status: response.status });
      } catch {
        return jsonError(
          response.ok ? 502 : response.status,
          "INVALID_RESELLER_API_RESPONSE",
          text || "Reseller API mengembalikan JSON tidak valid",
        );
      }
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
    return jsonError(502, "RESELLER_API_ERROR", error instanceof Error ? error.message : "Reseller API request failed");
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
