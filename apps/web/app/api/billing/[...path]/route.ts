import { NextRequest, NextResponse } from "next/server";
import { billingApi } from "../../../lib/billing-api";

type RouteContext = {
  params: Promise<{ path: string[] }>;
};

async function proxy(request: NextRequest, context: RouteContext) {
  try {
    const { path } = await context.params;
    const query = request.nextUrl.search || "";
    const targetPath = `/api/v1/${path.join("/")}${query}`;
    const method = request.method;
    const hasBody = !["GET", "HEAD"].includes(method);
    const data = await billingApi(targetPath, {
      method,
      body: hasBody ? await request.text() : undefined,
    });

    return NextResponse.json(data);
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
