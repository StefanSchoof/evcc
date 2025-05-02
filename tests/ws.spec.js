import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test("show no config screen while startup", async ({ page }) => {
  await page.routeWebSocket("/ws", () => {
    // connect, but don't send any messages
  });
  await page.goto("/");
  await expect(page.getByTestId("savings-button")).toBeVisible(); // ensure the page is loaded
  await expect(page.getByRole("link", { name: "Let's start configuration" })).toBeHidden();
});
