package webindexer

import (
	"fmt"
)

type Config struct {
	BaseURL        string   `mapstructure:"base_url"`
	DateFormat     string   `mapstructure:"date_format"`
	DirsFirst      bool     `mapstructure:"dirs_first"`
	IndexFile      string   `mapstructure:"index_file"`
	LinkToIndexes  bool     `mapstructure:"link_to_index"`
	LinkUpFromRoot bool     `mapstructure:"link_up_from_root"`
	LinkUpText     string   `mapstructure:"link_up_text"`
	LinkUpURL      string   `mapstructure:"link_up_url"`
	LogLevel       string   `mapstructure:"log_level"`
	LogFile        string   `mapstructure:"log_file"`
	Minify         bool     `mapstructure:"minify"`
	NoIndexFiles   []string `mapstructure:"noindex_files"`
	SkipIndexFiles []string `mapstructure:"skipindex_files"`
	Order          string   `mapstructure:"order"`
	Quiet          bool     `mapstructure:"quiet"`
	Recursive      bool     `mapstructure:"recursive"`
	Skips          []string `mapstructure:"skips"`
	SortBy         string   `mapstructure:"sort_by"`
	Source         string   `mapstructure:"source"`
	Target         string   `mapstructure:"target"`
	Template       string   `mapstructure:"template"`
	Theme          string   `mapstructure:"theme"`
	Title          string   `mapstructure:"title"`
	CfgFile        string   `mapstructure:"-"`
	BasePath       string   `mapstructure:"-"`
	S3Endpoint     string   `mapstructure:"s3_endpoint"`
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
