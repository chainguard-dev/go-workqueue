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

  title  = "Work In Progress"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_in_progress_keys/gauge\"",
    "metric.label.\"service_name\"=\"${var.name}-dsp\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align   = "ALIGN_MAX"
  primary_reduce  = "REDUCE_MAX"
}

module "work-queued" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy"
  version = "0.6.65"

  title  = "Work Queued"
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

  title  = "Work Added"
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

module "work-deduped" {
  source  = "chainguard-dev/common/infra//modules/dashboard/widgets/xy"
  version = "0.6.65"

  title  = "Work Deduplicated"
  filter = [
    "resource.type=\"prometheus_target\"",
    "metric.type=\"prometheus.googleapis.com/workqueue_deduped_keys_total/counter\"",
    "metric.label.\"service_name\"=\"${var.name}-rcv\"",
  ]
  group_by_fields = ["metric.label.\"service_name\""]
  primary_align    = "ALIGN_RATE"
  primary_reduce   = "REDUCE_NONE"
  secondary_align  = "ALIGN_NONE"
  secondary_reduce = "REDUCE_SUM"
}

locals {
  columns = 2
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
      yPos   = local.unit,
      xPos   = local.col[0],
      height = local.unit,
      width  = local.unit,
      widget = module.work-added.widget,
    },
    {
      yPos   = local.unit,
      xPos   = local.col[1],
      height = local.unit,
      width  = local.unit,
      widget = module.work-deduped.widget,
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
