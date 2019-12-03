// +build go1.13

package crontask

const cronErrorFormat = "error while executing synchronized task function %q: %w"
const renewalErrorFormat = "failed to renew leadership for synchronized task %q lock while executing: %w - crudely canceling"
