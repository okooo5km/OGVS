// Copyright (c) 2026 okooo5km(十里)
// SPDX-License-Identifier: MIT

package cli_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/okooo5km/ogvs/internal/version"
)

// binaryPath is the path to the built ogvs binary, shared across all tests.
var binaryPath string

func TestMain(m *testing.M) {
	// Build binary to a temp directory
	dir, err := os.MkdirTemp("", "ogvs-cli-test")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	binaryPath = filepath.Join(dir, "ogvs")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/ogvs")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// ---------- helpers ----------

// runOgvs runs the ogvs binary with args. Returns stdout, stderr, and error.
func runOgvs(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// writeTempFile creates a file in dir with the given name and content.
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// simpleSVG is a small SVG with a comment and empty group — easy to verify optimization.
const simpleSVG = `<svg xmlns="http://www.w3.org/2000/svg"><!-- comment --><g/></svg>`

// expectedSimpleSVG is the expected output after preset-default optimization.
const expectedSimpleSVG = `<svg xmlns="http://www.w3.org/2000/svg"/>`

// ---------- tests ----------

func TestCLI_Version(t *testing.T) {
	stdout, _, err := runOgvs(t, "", "--version")
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	// The output should contain the version string (from internal/version/version.go).
	if !strings.Contains(stdout, version.Version) {
		t.Errorf("expected version output to contain %q, got: %q", version.Version, stdout)
	}
}

func TestCLI_ShowPlugins(t *testing.T) {
	stdout, _, err := runOgvs(t, "", "--show-plugins")
	if err != nil {
		t.Fatalf("--show-plugins failed: %v", err)
	}
	// Should list plugin names and mention "preset-default"
	if !strings.Contains(stdout, "preset-default") {
		t.Errorf("expected output to contain 'preset-default', got: %q", stdout)
	}
	// Spot-check a few well-known plugins
	for _, name := range []string{"removeComments", "removeDoctype", "convertPathData", "cleanupIds"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected output to contain plugin %q", name)
		}
	}
}

func TestCLI_StringToStdout(t *testing.T) {
	stdout, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", "-")
	if err != nil {
		t.Fatalf("-s ... -o - failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_StdinPipeToStdout(t *testing.T) {
	// Piping via stdin with "-" as input, "-o -" for stdout
	stdout, _, err := runOgvs(t, simpleSVG, "-", "-o", "-")
	if err != nil {
		t.Fatalf("stdin pipe failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_StdinPipeImplicit(t *testing.T) {
	// When stdin is piped and no args are given, ogvs should read from stdin.
	// We simulate this with "-i -" to explicitly request stdin.
	stdout, _, err := runOgvs(t, simpleSVG, "-i", "-", "-o", "-")
	if err != nil {
		t.Fatalf("stdin implicit pipe failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_FileToFile(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)
	outFile := filepath.Join(dir, "output.svg")

	_, _, err := runOgvs(t, "", inFile, "-o", outFile)
	if err != nil {
		t.Fatalf("file-to-file failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_FileOverwrite(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)

	// No -o flag: should overwrite in place
	_, _, err := runOgvs(t, "", inFile)
	if err != nil {
		t.Fatalf("file overwrite failed: %v", err)
	}

	data, err := os.ReadFile(inFile)
	if err != nil {
		t.Fatalf("failed to read overwritten file: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_FileToStdout(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)

	stdout, _, err := runOgvs(t, "", inFile, "-o", "-")
	if err != nil {
		t.Fatalf("file-to-stdout failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_FolderRecursive(t *testing.T) {
	dir := t.TempDir()
	inDir := filepath.Join(dir, "input")
	outDir := filepath.Join(dir, "output")

	// Create folder structure with SVG files at different levels
	writeTempFile(t, inDir, "root.svg", simpleSVG)
	writeTempFile(t, inDir, "sub/nested.svg", simpleSVG)
	writeTempFile(t, inDir, "sub/deep/deep.svg", simpleSVG)
	// Non-SVG file should be skipped
	writeTempFile(t, inDir, "readme.txt", "not an svg")

	_, _, err := runOgvs(t, "", "-f", inDir, "-o", outDir, "-r")
	if err != nil {
		t.Fatalf("folder recursive failed: %v", err)
	}

	// Verify all SVG files were processed
	for _, relPath := range []string{"root.svg", "sub/nested.svg", "sub/deep/deep.svg"} {
		outPath := filepath.Join(outDir, relPath)
		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Errorf("missing output file %s: %v", relPath, err)
			continue
		}
		got := strings.TrimSpace(string(data))
		if got != expectedSimpleSVG {
			t.Errorf("%s: expected %q, got %q", relPath, expectedSimpleSVG, got)
		}
	}

	// Verify non-SVG file was NOT copied
	txtPath := filepath.Join(outDir, "readme.txt")
	if _, err := os.Stat(txtPath); err == nil {
		t.Error("non-SVG file should not have been processed")
	}
}

func TestCLI_FolderNonRecursive(t *testing.T) {
	dir := t.TempDir()
	inDir := filepath.Join(dir, "input")
	outDir := filepath.Join(dir, "output")

	writeTempFile(t, inDir, "root.svg", simpleSVG)
	writeTempFile(t, inDir, "sub/nested.svg", simpleSVG)

	// Without -r, subdirectories should be skipped
	_, _, err := runOgvs(t, "", "-f", inDir, "-o", outDir)
	if err != nil {
		t.Fatalf("folder non-recursive failed: %v", err)
	}

	// Root file should be processed
	data, err := os.ReadFile(filepath.Join(outDir, "root.svg"))
	if err != nil {
		t.Fatal("root.svg not found in output")
	}
	got := strings.TrimSpace(string(data))
	if got != expectedSimpleSVG {
		t.Errorf("root.svg: expected %q, got %q", expectedSimpleSVG, got)
	}

	// Nested file should NOT be processed
	if _, err := os.Stat(filepath.Join(outDir, "sub/nested.svg")); err == nil {
		t.Error("nested file should not have been processed without -r flag")
	}
}

func TestCLI_FolderExclude(t *testing.T) {
	dir := t.TempDir()
	inDir := filepath.Join(dir, "input")
	outDir := filepath.Join(dir, "output")

	writeTempFile(t, inDir, "keep.svg", simpleSVG)
	writeTempFile(t, inDir, "skip_this.svg", simpleSVG)

	_, _, err := runOgvs(t, "", "-f", inDir, "-o", outDir, "--exclude", "skip_")
	if err != nil {
		t.Fatalf("folder exclude failed: %v", err)
	}

	// keep.svg should be present
	if _, err := os.ReadFile(filepath.Join(outDir, "keep.svg")); err != nil {
		t.Error("keep.svg should have been processed")
	}

	// skip_this.svg should be excluded
	if _, err := os.Stat(filepath.Join(outDir, "skip_this.svg")); err == nil {
		t.Error("skip_this.svg should have been excluded")
	}
}

func TestCLI_PrettyIndent(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="100"/></svg>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--pretty", "--indent", "2")
	if err != nil {
		t.Fatalf("--pretty --indent failed: %v", err)
	}

	// Pretty output should have newlines and indentation
	if !strings.Contains(stdout, "\n") {
		t.Error("pretty output should contain newlines")
	}
	// Check that indent is 2 spaces (rect should be indented)
	if !strings.Contains(stdout, "  <") {
		t.Errorf("expected 2-space indentation in output: %q", stdout)
	}
}

func TestCLI_Multipass(t *testing.T) {
	// Use a more complex SVG where multipass might differ from single pass
	input := `<svg xmlns="http://www.w3.org/2000/svg"><!-- comment --><g><g><rect width="100" height="100"/></g></g></svg>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--multipass")
	if err != nil {
		t.Fatalf("--multipass failed: %v", err)
	}

	// Output should be valid and optimized (at minimum, comment should be removed)
	got := strings.TrimSpace(stdout)
	if strings.Contains(got, "<!--") {
		t.Error("multipass output should have removed comments")
	}
	// Should be non-empty
	if len(got) == 0 {
		t.Error("multipass output is empty")
	}
}

func TestCLI_EolCrlf(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="100"/></svg>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--pretty", "--eol", "crlf")
	if err != nil {
		t.Fatalf("--eol crlf failed: %v", err)
	}

	// Output should contain \r\n line endings
	if !strings.Contains(stdout, "\r\n") {
		t.Errorf("expected CRLF line endings in output: %q", stdout)
	}
}

func TestCLI_FinalNewline(t *testing.T) {
	stdout, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", "-", "--final-newline")
	if err != nil {
		t.Fatalf("--final-newline failed: %v", err)
	}

	// Output should end with a newline
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("expected output to end with newline, got: %q", stdout)
	}
}

func TestCLI_EolCrlfFinalNewline(t *testing.T) {
	input := `<svg xmlns="http://www.w3.org/2000/svg"><rect width="100" height="100"/></svg>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--pretty", "--eol", "crlf", "--final-newline")
	if err != nil {
		t.Fatalf("--eol crlf --final-newline failed: %v", err)
	}

	// Should have CRLF line endings
	if !strings.Contains(stdout, "\r\n") {
		t.Errorf("expected CRLF line endings in output: %q", stdout)
	}
	// Should end with newline
	if !strings.HasSuffix(stdout, "\r\n") {
		t.Errorf("expected output to end with CRLF, got suffix: %q", stdout[max(0, len(stdout)-20):])
	}
}

func TestCLI_DataURIBase64(t *testing.T) {
	// CRITICAL: Write output to a file, do NOT capture data URI in stdout.
	dir := t.TempDir()
	outFile := filepath.Join(dir, "output.txt")

	_, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", outFile, "--datauri", "base64")
	if err != nil {
		t.Fatalf("--datauri base64 failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	prefix := "data:image/svg+xml;base64,"
	if !strings.HasPrefix(content, prefix) {
		// Only show the first 60 chars to avoid dumping large base64
		show := content
		if len(show) > 60 {
			show = show[:60] + "..."
		}
		t.Errorf("expected output to start with %q, got: %q", prefix, show)
	}
}

func TestCLI_DataURIEnc(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "output.txt")

	_, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", outFile, "--datauri", "enc")
	if err != nil {
		t.Fatalf("--datauri enc failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	prefix := "data:image/svg+xml,"
	if !strings.HasPrefix(content, prefix) {
		show := content
		if len(show) > 80 {
			show = show[:80] + "..."
		}
		t.Errorf("expected output to start with %q, got: %q", prefix, show)
	}
}

func TestCLI_DataURIUnenc(t *testing.T) {
	dir := t.TempDir()
	outFile := filepath.Join(dir, "output.txt")

	_, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", outFile, "--datauri", "unenc")
	if err != nil {
		t.Fatalf("--datauri unenc failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	content := string(data)
	prefix := "data:image/svg+xml,"
	if !strings.HasPrefix(content, prefix) {
		show := content
		if len(show) > 80 {
			show = show[:80] + "..."
		}
		t.Errorf("expected output to start with %q, got: %q", prefix, show)
	}
}

func TestCLI_Config(t *testing.T) {
	dir := t.TempDir()

	// Create a config file
	configContent := `multipass: true
plugins:
  - preset-default
`
	configFile := writeTempFile(t, dir, "ogvs.config.yaml", configContent)

	stdout, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", "-", "--config", configFile)
	if err != nil {
		t.Fatalf("--config failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_ConfigWithPluginParams(t *testing.T) {
	dir := t.TempDir()

	// Config that disables removeComments via preset-default overrides
	// SVGO format: params.overrides.<pluginName>: false
	configContent := `plugins:
  - name: preset-default
    params:
      overrides:
        removeComments: false
`
	configFile := writeTempFile(t, dir, "ogvs.config.yaml", configContent)

	input := `<svg xmlns="http://www.w3.org/2000/svg"><!-- keep me --></svg>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--config", configFile)
	if err != nil {
		t.Fatalf("--config with plugin params failed: %v", err)
	}
	// Comments should be preserved since removeComments is disabled.
	// Note: other plugins (e.g. cleanupAttrs) may normalize whitespace inside the comment.
	if !strings.Contains(stdout, "<!--") {
		t.Errorf("expected comment to be preserved when removeComments=false, got: %q", stdout)
	}
	if !strings.Contains(stdout, "keep me") {
		t.Errorf("expected comment content to be preserved, got: %q", stdout)
	}
}

func TestCLI_ConfigDiscovery(t *testing.T) {
	dir := t.TempDir()

	// Create a config file in the "project" root
	configContent := `plugins:
  - name: preset-default
    params:
      overrides:
        removeComments: false
`
	writeTempFile(t, dir, "ogvs.config.yaml", configContent)

	// Create an input file in a subdirectory
	input := `<svg xmlns="http://www.w3.org/2000/svg"><!-- keep me --></svg>`
	inFile := writeTempFile(t, filepath.Join(dir, "subdir"), "input.svg", input)
	outFile := filepath.Join(dir, "subdir", "output.svg")

	// Run ogvs from the subdirectory — config discovery should find ogvs.config.yaml
	cmd := exec.Command(binaryPath, inFile, "-o", outFile)
	cmd.Dir = filepath.Join(dir, "subdir")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config discovery failed: %v\noutput: %s", err, string(out))
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	// Comment should be preserved since discovered config disables removeComments.
	// Note: other plugins may normalize whitespace inside the comment.
	output := string(data)
	if !strings.Contains(output, "<!--") {
		t.Errorf("expected discovered config to preserve comments, got: %q", output)
	}
	if !strings.Contains(output, "keep me") {
		t.Errorf("expected comment content to be preserved, got: %q", output)
	}
}

func TestCLI_ErrorInvalidFile(t *testing.T) {
	_, stderr, err := runOgvs(t, "", "/nonexistent/path/file.svg")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
	// Should have a meaningful error message
	if !strings.Contains(stderr, "Error") && !strings.Contains(stderr, "no such file") {
		t.Errorf("expected error message about non-existent file, got stderr: %q", stderr)
	}
}

func TestCLI_ErrorInvalidPrecision(t *testing.T) {
	_, stderr, err := runOgvs(t, "", "-s", simpleSVG, "--precision", "99")
	if err == nil {
		t.Fatal("expected error for invalid precision, got nil")
	}
	if !strings.Contains(stderr, "precision") {
		t.Errorf("expected error about precision, got: %q", stderr)
	}
}

func TestCLI_ErrorInvalidEol(t *testing.T) {
	_, stderr, err := runOgvs(t, "", "-s", simpleSVG, "--eol", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid eol, got nil")
	}
	if !strings.Contains(stderr, "eol") {
		t.Errorf("expected error about eol, got: %q", stderr)
	}
}

func TestCLI_ErrorInvalidDataURI(t *testing.T) {
	_, stderr, err := runOgvs(t, "", "-s", simpleSVG, "--datauri", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid datauri, got nil")
	}
	if !strings.Contains(stderr, "datauri") {
		t.Errorf("expected error about datauri, got: %q", stderr)
	}
}

func TestCLI_Quiet(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)
	outFile := filepath.Join(dir, "output.svg")

	_, stderr, err := runOgvs(t, "", inFile, "-o", outFile, "--quiet")
	if err != nil {
		t.Fatalf("--quiet failed: %v", err)
	}
	// With --quiet, stderr should have no stats output
	if strings.Contains(stderr, "Done in") {
		t.Errorf("--quiet should suppress stats output, got stderr: %q", stderr)
	}
	if strings.Contains(stderr, "KiB") {
		t.Errorf("--quiet should suppress stats output, got stderr: %q", stderr)
	}
}

func TestCLI_QuietStillWrites(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)
	outFile := filepath.Join(dir, "output.svg")

	_, _, err := runOgvs(t, "", inFile, "-o", outFile, "-q")
	if err != nil {
		t.Fatalf("-q failed: %v", err)
	}

	// Even with quiet, output file should still be written
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal("output file should still be written with --quiet")
	}
	got := strings.TrimSpace(string(data))
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_Precision(t *testing.T) {
	// SVG with numeric values that will be affected by precision
	input := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100.123456789 200.987654321"><rect width="100.123456789" height="200.987654321"/></svg>`

	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-", "--precision", "2")
	if err != nil {
		t.Fatalf("--precision failed: %v", err)
	}

	// With precision=2, long decimal values should be shortened
	got := strings.TrimSpace(stdout)
	if strings.Contains(got, "123456789") {
		t.Errorf("precision=2 should truncate long decimals, got: %q", got)
	}
}

func TestCLI_MultipleInputFiles(t *testing.T) {
	dir := t.TempDir()
	inFile1 := writeTempFile(t, dir, "a.svg", simpleSVG)
	inFile2 := writeTempFile(t, dir, "b.svg", simpleSVG)
	outFile1 := filepath.Join(dir, "out_a.svg")
	outFile2 := filepath.Join(dir, "out_b.svg")

	_, _, err := runOgvs(t, "", inFile1, inFile2, "-o", outFile1+","+outFile2)
	if err != nil {
		t.Fatalf("multiple inputs failed: %v", err)
	}

	// Both output files should be optimized
	for _, p := range []string{outFile1, outFile2} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("missing output %s: %v", p, err)
			continue
		}
		got := strings.TrimSpace(string(data))
		if got != expectedSimpleSVG {
			t.Errorf("%s: expected %q, got %q", p, expectedSimpleSVG, got)
		}
	}
}

func TestCLI_InputFlag(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)

	stdout, _, err := runOgvs(t, "", "-i", inFile, "-o", "-")
	if err != nil {
		t.Fatalf("-i flag failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	if got != expectedSimpleSVG {
		t.Errorf("expected %q, got %q", expectedSimpleSVG, got)
	}
}

func TestCLI_StatsOutput(t *testing.T) {
	dir := t.TempDir()
	inFile := writeTempFile(t, dir, "input.svg", simpleSVG)
	outFile := filepath.Join(dir, "output.svg")

	_, stderr, err := runOgvs(t, "", inFile, "-o", outFile)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	// When writing to a file (not stdout), stats should appear on stderr
	if !strings.Contains(stderr, "Done in") {
		t.Errorf("expected stats on stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "KiB") {
		t.Errorf("expected size info in stats, got: %q", stderr)
	}
}

func TestCLI_NoStatsToStdout(t *testing.T) {
	// When output goes to stdout (-o -), stats should not appear on stdout
	stdout, _, err := runOgvs(t, "", "-s", simpleSVG, "-o", "-")
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if strings.Contains(stdout, "Done in") {
		t.Error("stats should not appear in stdout when using -o -")
	}
	if strings.Contains(stdout, "KiB") {
		t.Error("size info should not appear in stdout when using -o -")
	}
}

func TestCLI_HelpWithNoArgs(t *testing.T) {
	// When no args and no stdin pipe, should show help
	// NOTE: We cannot truly simulate "no pipe" from exec.Command because
	// exec.Command doesn't have a tty. Instead, we just check that
	// running with no args doesn't crash.
	cmd := exec.Command(binaryPath)
	// Set stdin to /dev/null to simulate no pipe
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	// Should exit 0 and show usage
	if err != nil {
		// Some CLI frameworks exit 0 for help, others exit 1
		// Just make sure it doesn't crash
		t.Logf("no-args exited with: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "ogvs") {
		t.Errorf("expected help output to mention 'ogvs', got: %q", output)
	}
}

func TestCLI_EmptySVG(t *testing.T) {
	// Edge case: empty SVG element
	input := `<svg xmlns="http://www.w3.org/2000/svg"/>`
	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-")
	if err != nil {
		t.Fatalf("empty SVG failed: %v", err)
	}
	got := strings.TrimSpace(stdout)
	// Should remain unchanged or minimally changed
	if !strings.Contains(got, "<svg") {
		t.Errorf("expected svg element in output, got: %q", got)
	}
}

func TestCLI_LargeSVG(t *testing.T) {
	// Construct a moderately large SVG to check performance doesn't break
	var sb strings.Builder
	sb.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000 1000">`)
	for i := 0; i < 100; i++ {
		sb.WriteString(`<!-- comment -->`)
		sb.WriteString(`<rect x="0" y="0" width="10" height="10"/>`)
	}
	sb.WriteString(`</svg>`)
	input := sb.String()

	stdout, _, err := runOgvs(t, "", "-s", input, "-o", "-")
	if err != nil {
		t.Fatalf("large SVG failed: %v", err)
	}

	got := strings.TrimSpace(stdout)
	// Comments should be removed
	if strings.Contains(got, "<!--") {
		t.Error("large SVG: comments should be removed")
	}
	// Output should be shorter than input
	if len(got) >= len(input) {
		t.Errorf("large SVG: output (%d bytes) should be shorter than input (%d bytes)", len(got), len(input))
	}
}

func TestCLI_FolderOverwrite(t *testing.T) {
	dir := t.TempDir()
	inDir := filepath.Join(dir, "svgs")

	writeTempFile(t, inDir, "a.svg", simpleSVG)
	writeTempFile(t, inDir, "b.svg", simpleSVG)

	// No -o: files should be overwritten in place
	_, _, err := runOgvs(t, "", "-f", inDir, "-q")
	if err != nil {
		t.Fatalf("folder overwrite failed: %v", err)
	}

	for _, name := range []string{"a.svg", "b.svg"} {
		data, err := os.ReadFile(filepath.Join(inDir, name))
		if err != nil {
			t.Errorf("missing file %s: %v", name, err)
			continue
		}
		got := strings.TrimSpace(string(data))
		if got != expectedSimpleSVG {
			t.Errorf("%s: expected %q, got %q", name, expectedSimpleSVG, got)
		}
	}
}

func TestCLI_ErrorInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := writeTempFile(t, dir, "bad.yaml", "{{{{invalid yaml")

	_, stderr, err := runOgvs(t, "", "-s", simpleSVG, "--config", configFile)
	if err == nil {
		t.Fatal("expected error for invalid config file, got nil")
	}
	if !strings.Contains(stderr, "Error") {
		t.Errorf("expected error message for invalid config, got: %q", stderr)
	}
}

func TestCLI_ErrorMissingConfig(t *testing.T) {
	_, stderr, err := runOgvs(t, "", "-s", simpleSVG, "--config", "/nonexistent/ogvs.config.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
	if !strings.Contains(stderr, "Error") {
		t.Errorf("expected error message for missing config, got: %q", stderr)
	}
}
