module "width" {
  source  = "chainguard-dev/common/infra//modules/dashboard/sections/width"
  version = "0.6.65"
}

// TODO: workqueue metrics.

module "receiver-logs" {
  source  = "chainguard-dev/common/infra//modules/dashboard/sections/logs"
  version = "0.6.65"

  title  = "Receiver Logs"
  filter = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}-rcv\""]
}

module "dispatcher-logs" {
  source  = "chainguard-dev/common/infra//modules/dashboard/sections/logs"
  version = "0.6.65"

  title  = "Dispatcher Logs"
  filter = ["resource.type=\"cloud_run_revision\"", "resource.labels.service_name=\"${var.name}-dsp\""]
}

module "work-in-progress" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy"
  version = "0.6.65"

  title  = "Amount of work in progress"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_in_progress_keys/gauge\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"

  # TODO(mattmoor): Add threshold when it lands in a release.
  # thresholds = [var.concurrent-work]
}

module "work-queued" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy"
  version = "0.6.65"

  title  = "Amount of work queued"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_queued_keys/gauge\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
}

module "work-added" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy"
  version = "0.6.65"

  title  = "Amount of work added"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

module "process-latency" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/latency"
  version = "0.6.65"

  title  = "Work processing latency"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_process_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
}

module "wait-latency" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/latency"
  version = "0.6.65"

  title  = "Work wait times"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_wait_latency_seconds/histogram\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
}

module "percent-deduped" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy-ratio"
  version = "0.6.65"

  title     = "Percentage of work deduplicated"
  legend    = ""
  plot_type = "LINE"

  numerator_filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_deduped_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]
  denominator_filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_added_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]

  alignment_period            = "60s"
  thresholds                  = []
  numerator_align             = "ALIGN_RATE"
  numerator_group_by_fields   = ["metric.label.\"service_name\""]
  numerator_reduce            = "REDUCE_SUM"
  denominator_align           = "ALIGN_RATE"
  denominator_group_by_fields = ["metric.label.\"service_name\""]
  denominator_reduce          = "REDUCE_SUM"
}

locals {
  columns = 3
  unit    = module.width.size / local.columns

  // https://www.terraform.io/language/functions/range
  // N columns, unit width each  ([0, unit, 2 * unit, ...])
  col = range(0, local.columns * local.unit, local.unit)

  tiles = [
    {
      yPos   = 0,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.work-in-progress.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.work-queued.widget,
    },
    {
      yPos   = 0,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.work-added.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.process-latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.wait-latency.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[2],
      height = local.unit,
      width  = local.unit,
      widget = module.percent-deduped.widget,
    },
  ]
}

module "collapsible" {
  source  = "chainguard-dev/common/infra//modules/dashboard/sections/collapsible"
  version = "0.6.65"

  title     = "Workqueue State"
  tiles     = local.tiles
  collapsed = false
}

module "layout" {
  source  = "chainguard-dev/common/infra//modules/dashboard/sections/layout"
  version = "0.6.65"

  sections = [
    module.collapsible.section,
    module.receiver-logs.section,
    module.dispatcher-logs.section,
  ]
}

resource "google_monitoring_dashboard" "dashboard" {
  dashboard_json = jsonencode({
    displayName = "Cloud Workqueue: ${var.name}"
    labels = {
      "service" : ""
      "workqueue" : ""
    }

    // https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards#mosaiclayout
    mosaicLayout = {
      columns = module.width.size
      tiles   = module.layout.tiles,
    }
  })
}
