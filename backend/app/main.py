"""
DevSecOps Demo — FastAPI Backend
Full CRUD Item Manager with OpenTelemetry instrumentation.
"""

from __future__ import annotations

import logging
import os
import sys
from contextlib import asynccontextmanager
from typing import Annotated

from fastapi import Depends, FastAPI, HTTPException, Query
from fastapi.middleware.cors import CORSMiddleware
from sqlmodel import Field, Session, SQLModel, create_engine, select

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
DATABASE_URL: str = os.getenv("DATABASE_URL", "sqlite:///./app.db")
ALLOWED_ORIGIN: str = os.getenv("ALLOWED_ORIGIN", "http://localhost:8080")
ENABLE_OBSERVABILITY: bool = os.getenv("ENABLE_OBSERVABILITY", "false").lower() == "true"
OTEL_ENDPOINT: str = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://alloy:4317")
PYROSCOPE_SERVER: str = os.getenv("PYROSCOPE_SERVER", "http://alloy:4040")
PORT: int = int(os.getenv("PORT", "8000"))

SERVICE_NAME = "backend"
SERVICE_VERSION = "1.0.0"

logger = logging.getLogger(SERVICE_NAME)

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------
connect_args = {"check_same_thread": False} if DATABASE_URL.startswith("sqlite") else {}
engine = create_engine(DATABASE_URL, connect_args=connect_args)


class Item(SQLModel, table=True):
    """Item model — the core domain object."""

    id: int | None = Field(default=None, primary_key=True)
    title: str = Field(index=True)
    description: str | None = Field(default=None)


def get_db() -> Session:  # type: ignore[misc]
    """Yield a database session."""
    with Session(engine) as session:
        yield session


SessionDep = Annotated[Session, Depends(get_db)]

# ---------------------------------------------------------------------------
# OpenTelemetry Setup
# ---------------------------------------------------------------------------
tracer = None  # Will be set if observability is enabled
meter = None
items_created_counter = None


def _setup_observability() -> None:
    """Configure OpenTelemetry traces, metrics, logs, and Pyroscope profiling."""
    global tracer, meter, items_created_counter  # noqa: PLW0603

    from opentelemetry import metrics as otel_metrics
    from opentelemetry import trace
    from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
    from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
    from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
    from opentelemetry.instrumentation.logging import LoggingInstrumentor
    from opentelemetry.instrumentation.sqlalchemy import SQLAlchemyInstrumentor
    from opentelemetry.sdk.metrics import MeterProvider
    from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
    from opentelemetry.sdk.resources import Resource
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor
    from opentelemetry.sdk.trace.sampling import TraceIdRatioBased

    # --- OTel Logs ---
    from opentelemetry._logs import set_logger_provider
    from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
    from opentelemetry.sdk._logs import LoggerProvider
    from opentelemetry.sdk._logs.export import BatchLogRecordProcessor

    resource = Resource.create(
        {
            "service.name": SERVICE_NAME,
            "service.version": SERVICE_VERSION,
        }
    )

    # Traces — 10 % sampling
    trace_provider = TracerProvider(
        resource=resource,
        sampler=TraceIdRatioBased(0.1),
    )
    trace_provider.add_span_processor(
        BatchSpanProcessor(OTLPSpanExporter(endpoint=OTEL_ENDPOINT, insecure=True))
    )
    trace.set_tracer_provider(trace_provider)
    tracer = trace.get_tracer(SERVICE_NAME, SERVICE_VERSION)

    # Metrics
    metric_reader = PeriodicExportingMetricReader(
        OTLPMetricExporter(endpoint=OTEL_ENDPOINT, insecure=True),
        export_interval_millis=15_000,
    )
    meter_provider = MeterProvider(resource=resource, metric_readers=[metric_reader])
    otel_metrics.set_meter_provider(meter_provider)
    meter = otel_metrics.get_meter(SERVICE_NAME, SERVICE_VERSION)
    items_created_counter = meter.create_counter(
        name="items_created_total",
        description="Total number of items created",
        unit="1",
    )

    # Logs
    log_provider = LoggerProvider(resource=resource)
    log_provider.add_log_record_processor(
        BatchLogRecordProcessor(OTLPLogExporter(endpoint=OTEL_ENDPOINT, insecure=True))
    )
    set_logger_provider(log_provider)
    LoggingInstrumentor().instrument(set_logging_format=True)

    # Auto-instrumentation
    SQLAlchemyInstrumentor().instrument(engine=engine.sync_engine)

    # Pyroscope — Linux only
    if sys.platform == "linux":
        try:
            import pyroscope  # type: ignore[import-untyped]

            pyroscope.configure(
                application_name=SERVICE_NAME,
                server_address=PYROSCOPE_SERVER,
            )
            logger.info("Pyroscope profiling enabled → %s", PYROSCOPE_SERVER)
        except Exception:
            logger.warning("Pyroscope not available — skipping profiling")

    logger.info("OpenTelemetry instrumentation initialised (endpoint=%s)", OTEL_ENDPOINT)


# ---------------------------------------------------------------------------
# Application Lifespan
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):  # noqa: ARG001
    """Create tables on startup, teardown on shutdown."""
    SQLModel.metadata.create_all(engine)
    logger.info("Database tables created / verified")
    yield


# ---------------------------------------------------------------------------
# FastAPI App
# ---------------------------------------------------------------------------
app = FastAPI(
    title="DevSecOps Item Manager",
    version=SERVICE_VERSION,
    lifespan=lifespan,
)

# CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=[ALLOWED_ORIGIN],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# OTel auto-instrumentation for FastAPI (must be done after app creation)
if ENABLE_OBSERVABILITY:
    _setup_observability()

    from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

    FastAPIInstrumentor.instrument_app(
        app,
        excluded_urls="healthz,readyz,metrics",
    )

# ---------------------------------------------------------------------------
# Health Probes
# ---------------------------------------------------------------------------


@app.get("/healthz", tags=["health"])
def liveness():
    """Kubernetes liveness probe."""
    return {"status": "ok"}


@app.get("/readyz", tags=["health"])
def readiness(session: SessionDep):
    """Kubernetes readiness probe — verifies DB connectivity."""
    try:
        session.exec(select(1))  # type: ignore[call-overload]
        return {"status": "ready"}
    except Exception:
        raise HTTPException(status_code=503, detail="Database not ready")


# ---------------------------------------------------------------------------
# CRUD Endpoints
# ---------------------------------------------------------------------------


@app.post("/items", status_code=201, tags=["items"])
def create_item(item: Item, session: SessionDep):
    """Create a new item."""
    if tracer:
        from opentelemetry import trace

        span = trace.get_current_span()
        span.set_attribute("item.title", item.title)

    session.add(item)
    session.commit()
    session.refresh(item)

    if items_created_counter:
        items_created_counter.add(1, {"status": "success"})

    logger.info("Item created: id=%s title=%s", item.id, item.title)
    return item


@app.get("/items", tags=["items"])
def list_items(
    session: SessionDep,
    skip: int = Query(default=0, ge=0),
    limit: int = Query(default=100, ge=1, le=1000),
):
    """List items with pagination."""
    if tracer:
        with tracer.start_as_current_span("fetch_all_items_from_db") as span:
            span.set_attribute("db.skip", skip)
            span.set_attribute("db.limit", limit)
            items = session.exec(select(Item).offset(skip).limit(limit)).all()
    else:
        items = session.exec(select(Item).offset(skip).limit(limit)).all()

    return items


@app.put("/items/{item_id}", tags=["items"])
def update_item(item_id: int, item_update: Item, session: SessionDep):
    """Update an existing item."""
    db_item = session.get(Item, item_id)
    if not db_item:
        raise HTTPException(status_code=404, detail="Item not found")

    db_item.title = item_update.title
    db_item.description = item_update.description
    session.add(db_item)
    session.commit()
    session.refresh(db_item)

    logger.info("Item updated: id=%s", item_id)
    return db_item


@app.delete("/items/{item_id}", status_code=204, tags=["items"])
def delete_item(item_id: int, session: SessionDep):
    """Delete an item."""
    db_item = session.get(Item, item_id)
    if not db_item:
        raise HTTPException(status_code=404, detail="Item not found")

    session.delete(db_item)
    session.commit()
    logger.info("Item deleted: id=%s", item_id)


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import uvicorn

    uvicorn.run(
        "main:app",
        host="0.0.0.0",
        port=PORT,
        workers=int(os.getenv("WEB_CONCURRENCY", "4")),
        log_level="info",
    )
