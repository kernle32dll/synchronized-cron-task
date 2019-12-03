// +build !go1.13

package crontask

const cronErrorFormat = "error while executing synchronized task function %q: %s"
const renewalErrorFormat = "failed to renew leadership for synchronized task %q lock while executing: %s - crudely canceling"
