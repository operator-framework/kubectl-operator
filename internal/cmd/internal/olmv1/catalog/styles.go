package catalog

import "github.com/charmbracelet/lipgloss"

var CatalogNameColor = lipgloss.AdaptiveColor{Light: "#E791A9", Dark: "#B06482"}
var CatalogNameStyle = lipgloss.NewStyle().Foreground(CatalogNameColor).Bold(true).Padding(0, 1)

var SchemaNameColor = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
var SchemaNameStyle = lipgloss.NewStyle().Foreground(SchemaNameColor).Bold(true)

var PackageNameColor = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
var PackageNameStyle = lipgloss.NewStyle().Foreground(PackageNameColor).Italic(true)

var NameColor = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
var NameStyle = lipgloss.NewStyle().Foreground(NameColor)
