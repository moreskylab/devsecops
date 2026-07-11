/**
 * DevSecOps Demo — Frontend Application
 *
 * Vanilla TypeScript CRUD client with full OpenTelemetry browser instrumentation
 * (traces, metrics, logs) exported via OTLP/HTTP.
 */

import "./style.css";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------
interface Item {
  id: number;
  title: string;
  description: string | null;
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------
const ENABLE_TRACKING = import.meta.env.VITE_ENABLE_TRACKING !== "false";
const APP_VERSION = import.meta.env.VITE_APP_VERSION ?? "1.0.0";
const API_BASE = "/items";

// ---------------------------------------------------------------------------
// OpenTelemetry Setup
// ---------------------------------------------------------------------------
let tracer: any = null;
let durationHistogram: any = null;
let otelLogs: any = null;

async function initTelemetry(): Promise<void> {
  if (!ENABLE_TRACKING) return;

  try {
    const { WebTracerProvider } = await import(
      "@opentelemetry/sdk-trace-web"
    );
    const { BatchSpanProcessor } = await import("@opentelemetry/sdk-trace-base");
    const { OTLPTraceExporter } = await import(
      "@opentelemetry/exporter-trace-otlp-http"
    );
    const { Resource } = await import("@opentelemetry/resources");
    const {
      ATTR_SERVICE_NAME,
      ATTR_SERVICE_VERSION,
    } = await import("@opentelemetry/semantic-conventions");
    const { ZoneContextManager } = await import("@opentelemetry/context-zone");
    const { registerInstrumentations } = await import(
      "@opentelemetry/instrumentation"
    );
    const { FetchInstrumentation } = await import(
      "@opentelemetry/instrumentation-fetch"
    );
    const { UserInteractionInstrumentation } = await import(
      "@opentelemetry/instrumentation-user-interaction"
    );
    const { trace } = await import("@opentelemetry/api");

    // Metrics
    const { MeterProvider, PeriodicExportingMetricReader } = await import(
      "@opentelemetry/sdk-metrics"
    );
    const { OTLPMetricExporter } = await import(
      "@opentelemetry/exporter-metrics-otlp-http"
    );

    // Logs
    const { LoggerProvider, BatchLogRecordProcessor } = await import(
      "@opentelemetry/sdk-logs"
    );
    const { OTLPLogExporter } = await import(
      "@opentelemetry/exporter-logs-otlp-http"
    );

    const resource = new Resource({
      [ATTR_SERVICE_NAME]: "frontend",
      [ATTR_SERVICE_VERSION]: APP_VERSION,
    });

    // --- Traces ---
    const traceExporter = new OTLPTraceExporter({
      url: "/v1/traces",
    });
    const provider = new WebTracerProvider({
      resource,
      spanProcessors: [new BatchSpanProcessor(traceExporter)],
    });
    provider.register({ contextManager: new ZoneContextManager() });

    registerInstrumentations({
      instrumentations: [
        new FetchInstrumentation({
          propagateTraceHeaderCorsUrls: [/.*/],
        }),
        new UserInteractionInstrumentation(),
      ],
    });

    tracer = trace.getTracer("frontend", APP_VERSION);

    // --- Metrics ---
    const metricExporter = new OTLPMetricExporter({
      url: "/v1/metrics",
    });
    const meterProvider = new MeterProvider({
      resource,
      readers: [
        new PeriodicExportingMetricReader({
          exporter: metricExporter,
          exportIntervalMillis: 15_000,
        }),
      ],
    });
    const meter = meterProvider.getMeter("frontend", APP_VERSION);
    durationHistogram = meter.createHistogram("ui.operation.duration", {
      description: "Duration of UI operations in milliseconds",
      unit: "ms",
    });

    // --- Logs ---
    const logExporter = new OTLPLogExporter({
      url: "/v1/logs",
    });
    const loggerProvider = new LoggerProvider({ resource });
    loggerProvider.addLogRecordProcessor(
      new BatchLogRecordProcessor(logExporter)
    );
    otelLogs = loggerProvider.getLogger("frontend", APP_VERSION);

    console.info("[OTel] Telemetry initialised");
  } catch (err) {
    console.warn("[OTel] Failed to initialise telemetry:", err);
  }
}

// ---------------------------------------------------------------------------
// Telemetry Helpers
// ---------------------------------------------------------------------------
function withSpan<T>(name: string, fn: () => T): T {
  if (!tracer) return fn();
  return tracer.startActiveSpan(name, (span: any) => {
    try {
      const result = fn();
      if (result instanceof Promise) {
        return (result as Promise<any>)
          .then((v: T) => {
            span.end();
            return v;
          })
          .catch((err: Error) => {
            span.recordException(err);
            span.end();
            throw err;
          }) as T;
      }
      span.end();
      return result;
    } catch (err) {
      span.recordException(err as Error);
      span.end();
      throw err;
    }
  });
}

function recordDuration(operation: string, startMs: number, status: string) {
  if (durationHistogram) {
    durationHistogram.record(performance.now() - startMs, {
      operation,
      status,
    });
  }
}

// ---------------------------------------------------------------------------
// DOM References
// ---------------------------------------------------------------------------
const form = document.getElementById("item-form") as HTMLFormElement;
const titleInput = document.getElementById("item-title") as HTMLInputElement;
const descInput = document.getElementById(
  "item-description"
) as HTMLTextAreaElement;
const idInput = document.getElementById("item-id") as HTMLInputElement;
const submitBtn = document.getElementById("btn-submit") as HTMLButtonElement;
const cancelBtn = document.getElementById("btn-cancel") as HTMLButtonElement;
const itemList = document.getElementById("item-list") as HTMLDivElement;
const emptyState = document.getElementById("empty-state") as HTMLParagraphElement;

// ---------------------------------------------------------------------------
// API Helpers
// ---------------------------------------------------------------------------
async function apiFetch<T>(
  url: string,
  options?: RequestInit
): Promise<T | null> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (res.status === 204) return null;
  if (!res.ok) throw new Error(`API error: ${res.status} ${res.statusText}`);
  return res.json() as Promise<T>;
}

// ---------------------------------------------------------------------------
// CRUD Operations
// ---------------------------------------------------------------------------
async function fetchItems(): Promise<Item[]> {
  return withSpan("ui.fetch_items", async () => {
    const items = await apiFetch<Item[]>(API_BASE);
    return items ?? [];
  });
}

async function createItem(title: string, description: string | null) {
  const start = performance.now();
  return withSpan("ui.submit_item", async () => {
    try {
      await apiFetch<Item>(API_BASE, {
        method: "POST",
        body: JSON.stringify({ title, description }),
      });
      recordDuration("create", start, "success");
    } catch (err) {
      recordDuration("create", start, "error");
      throw err;
    }
  });
}

async function updateItem(
  id: number,
  title: string,
  description: string | null
) {
  const start = performance.now();
  return withSpan("ui.update_item", async () => {
    try {
      await apiFetch<Item>(`${API_BASE}/${id}`, {
        method: "PUT",
        body: JSON.stringify({ title, description }),
      });
      recordDuration("update", start, "success");
    } catch (err) {
      recordDuration("update", start, "error");
      throw err;
    }
  });
}

async function deleteItem(id: number) {
  const start = performance.now();
  return withSpan("ui.delete_item", async () => {
    try {
      await apiFetch(`${API_BASE}/${id}`, { method: "DELETE" });
      recordDuration("delete", start, "success");
    } catch (err) {
      recordDuration("delete", start, "error");
      throw err;
    }
  });
}

// ---------------------------------------------------------------------------
// Rendering (DocumentFragment for batch DOM updates)
// ---------------------------------------------------------------------------
function renderItems(items: Item[]): void {
  withSpan("ui.render_list", () => {
    if (items.length === 0) {
      itemList.innerHTML = "";
      emptyState.hidden = false;
      return;
    }

    emptyState.hidden = true;
    const fragment = document.createDocumentFragment();

    for (const item of items) {
      const card = document.createElement("div");
      card.className = "item-card";
      card.dataset.id = String(item.id);

      card.innerHTML = `
        <div class="item-content">
          <h3 class="item-title">${escapeHtml(item.title)}</h3>
          ${item.description ? `<p class="item-desc">${escapeHtml(item.description)}</p>` : ""}
        </div>
        <div class="item-actions">
          <button class="btn-edit" data-id="${item.id}" aria-label="Edit item ${item.id}">✏️</button>
          <button class="btn-delete" data-id="${item.id}" aria-label="Delete item ${item.id}">🗑️</button>
        </div>
      `;

      fragment.appendChild(card);
    }

    itemList.innerHTML = "";
    itemList.appendChild(fragment);
  });
}

function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// ---------------------------------------------------------------------------
// Form Handling
// ---------------------------------------------------------------------------
function resetForm(): void {
  form.reset();
  idInput.value = "";
  submitBtn.textContent = "Create Item";
  cancelBtn.hidden = true;
}

function populateForm(item: Item): void {
  idInput.value = String(item.id);
  titleInput.value = item.title;
  descInput.value = item.description ?? "";
  submitBtn.textContent = "Update Item";
  cancelBtn.hidden = false;
  titleInput.focus();
}

// ---------------------------------------------------------------------------
// Event Handlers (Event delegation)
// ---------------------------------------------------------------------------
async function refreshList(): Promise<void> {
  try {
    const items = await fetchItems();
    renderItems(items);
  } catch (err) {
    console.error("Failed to fetch items:", err);
  }
}

form.addEventListener("submit", async (e: Event) => {
  e.preventDefault();
  const title = titleInput.value.trim();
  const description = descInput.value.trim() || null;
  const editId = idInput.value;

  if (!title) return;

  try {
    if (editId) {
      await updateItem(Number(editId), title, description);
    } else {
      await createItem(title, description);
    }
    resetForm();
    await refreshList();
  } catch (err) {
    console.error("Form submission failed:", err);
  }
});

cancelBtn.addEventListener("click", () => {
  resetForm();
});

// Event delegation on the item list
itemList.addEventListener("click", async (e: Event) => {
  const target = e.target as HTMLElement;
  const btn = target.closest("button");
  if (!btn) return;

  const id = Number(btn.dataset.id);
  if (!id) return;

  if (btn.classList.contains("btn-delete")) {
    try {
      await deleteItem(id);
      await refreshList();
    } catch (err) {
      console.error("Delete failed:", err);
    }
  }

  if (btn.classList.contains("btn-edit")) {
    // Find the item data from the DOM
    const card = btn.closest(".item-card") as HTMLElement;
    const title =
      card.querySelector(".item-title")?.textContent ?? "";
    const desc =
      card.querySelector(".item-desc")?.textContent ?? "";
    populateForm({ id, title, description: desc || null });
  }
});

// ---------------------------------------------------------------------------
// Initialise
// ---------------------------------------------------------------------------
(async () => {
  await initTelemetry();
  await refreshList();
})();
