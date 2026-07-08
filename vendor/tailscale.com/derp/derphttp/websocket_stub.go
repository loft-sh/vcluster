// Copyright (c) Tailscale Inc & contributors
// SPDX-License-Identifier: BSD-3-Clause

//go:build !(linux || windows || (darwin && !ios) || !js)

package derphttp

const canWebsockets = false
