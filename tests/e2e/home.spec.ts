import { expect, test } from "@playwright/test";

test("homepage shows the public landing page", async ({ page }) => {
  await page.goto("/");

  await expect(
    page.getByRole("heading", {
      name: /ispboss kelola isp dari satu dashboard/i,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: /coba gratis 3 hari/i }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: /pertanyaan sebelum mulai/i }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Kontak" }),
  ).toBeVisible();
});
