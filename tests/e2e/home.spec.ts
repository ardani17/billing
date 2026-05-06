import { expect, test } from "@playwright/test";

test("homepage shows the main entry points", async ({ page }) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", {
      name: /platform billing dan manajemen jaringan untuk isp/i,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Landing Page" }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Tenant Dashboard" }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Super Admin" }),
  ).toBeVisible();
});
