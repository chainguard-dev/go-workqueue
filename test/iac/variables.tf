variable "namespace" {
  type = string
}

variable "name" {
  type = string
}

variable "concurrent-work" {
  description = "The amount of concurrent work to dispatch at a given time."
  type        = number
}

variable "reconciler-service" {
  description = "The address of the k8s service to push keys to."
  type        = string
}
