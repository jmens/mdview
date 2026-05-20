//go:build !linux

package main

// startStderrFilter is a no-op on platforms other than Linux. The Linux
// implementation filters WebKitGTK/JSC noise; WebView2 (Windows) and WKWebView
// (macOS) do not emit equivalent stderr chatter, so there is nothing to filter.
func startStderrFilter() (func(), error) {
	return func() {}, nil
}
