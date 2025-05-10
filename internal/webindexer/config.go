package webindexer

import (
	"fmt"
)

type Config struct {
	BaseURL        string   `yaml:"base_url"      mapstructure:"base_url"`
	DateFormat     string   `yaml:"date_format"   mapstructure:"date_format"`
	DirsFirst      bool     `yaml:"dirs_first"    mapstructure:"dirs_first"`
	IndexFile      string   `yaml:"index_file"    mapstructure:"index_file"`
	LinkToIndexes  bool     `yaml:"link_to_index" mapstructure:"link_to_index"`
	LogLevel       string   `yaml:"log_level"     mapstructure:"log_level"`
	LogFile        string   `yaml:"log_file"      mapstructure:"log_file"`
	Minify         bool     `yaml:"minify"        mapstructure:"minify"`
	NoIndexFiles   []string `yaml:"noindex_files" mapstructure:"noindex_files"`
	SkipIndexFiles []string `yaml:"skipindex_files" mapstructure:"skipindex_files"`
	Order          string   `yaml:"order"         mapstructure:"order"`
	Quiet          bool     `yaml:"quiet"         mapstructure:"quiet"`
	Recursive      bool     `yaml:"recursive"     mapstructure:"recursive"`
	Skips          []string `yaml:"skips"         mapstructure:"skips"`
	SortBy         string   `yaml:"sort_by"       mapstructure:"sort_by"`
	Source         string   `yaml:"source"        mapstructure:"source"`
	Target         string   `yaml:"target"        mapstructure:"target"`
	Template       string   `yaml:"template"      mapstructure:"template"`
	Theme          string   `yaml:"theme"         mapstructure:"theme"`
	Title          string   `yaml:"title"         mapstructure:"title"`
	CfgFile        string   `yaml:"-"`
	BasePath       string   `yaml:"-"`
	S3Endpoint     string   `yaml:"s3_endpoint"   mapstructure:"s3_endpoint"`
}

type SortBy string

const (
	SortByDate        SortBy = "date"
	SortByName        SortBy = "name"
	SortByNaturalName SortBy = "natural_name"
)

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

type Theme string

const (
	ThemeDefault   Theme = "default"
	ThemeSolarized Theme = "solarized"
	ThemeNord      Theme = "nord"
	ThemeDracula   Theme = "dracula"
)

func (c Config) SortByValue() SortBy {
	switch c.SortBy {
	case "last_modified":
		return SortByDate
	case "name":
		return SortByName
	case "natural_name":
		return SortByNaturalName
	default:
		return ""
	}
}

func (c Config) OrderByValue() Order {
	switch c.Order {
	case "asc":
		return OrderAsc
	case "desc":
		return OrderDesc
	default:
		return ""
	}
}

func (c Config) ThemeValue() Theme {
	switch c.Theme {
	case "solarized":
		return ThemeSolarized
	case "nord":
		return ThemeNord
	case "dracula":
		return ThemeDracula
	default:
		return ThemeDefault
	}
}

func (c Config) Validate() error {
	if c.Source == "" {
		return fmt.Errorf("source is required")
	}

	if c.Target == "" {
		return fmt.Errorf("target is required")
	}

	if c.SortByValue() == "" {
		return fmt.Errorf("sort_by must be one of: last_modified, name, natural_name")
	}

	if c.OrderByValue() == "" {
		return fmt.Errorf("order must be one of: asc, desc")
	}

	return nil
}
