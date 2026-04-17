# How to Use Configuration with Viper

**What is Viper?** Viper is a configuration management library that reads from multiple sources with precedence:

1. Command-line flags (highest priority)
2. Environment variables
3. Config files (YAML, JSON, TOML)
4. Default values (lowest priority)

## Table of Contents

- [Config Structure](#config-structure)
- [LoadConfig Function](#loadconfig-function)
- [Why mapstructure Tags Are Needed](#why-mapstructure-tags-are-needed)
- [Usage Examples](#usage-examples)

---

## Config Structure

Define your Config struct with all settings:

```go
type Config struct {
    Port       int    `mapstructure:"port"`
    DBHost     string `mapstructure:"db_host"`
    DBPassword string `mapstructure:"db_password"`
    // ... more fields
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## LoadConfig Function

The LoadConfig function does the magic:

```go
func LoadConfig() (*Config, error) {
    // Bind environment variables to keys
    viper.BindEnv("port", "PORT")
    viper.BindEnv("db_host", "DB_HOST")

    // Set defaults
    viper.SetDefault("port", 8080)
    viper.SetDefault("db_host", "localhost")

    // Define command-line flags (using pflag)
    pflag.Int("port", viper.GetInt("port"), "Server port")
    pflag.String("db_host", viper.GetString("db_host"), "Database host")
    pflag.Parse()

    // Bind flags to Viper
    viper.BindPFlags(pflag.CommandLine)

    // Unmarshal everything into Config struct
    var config Config
    viper.Unmarshal(&config)

    return &config, nil
}
```

**How it works:**
- `mapstructure` tags tell Viper which config key maps to which struct field
- `BindEnv` connects env vars (e.g., `PORT`) to Viper keys (e.g., `port`)
- `pflag` creates CLI flags that users can pass: `--port 3000`
- `Unmarshal` fills your Config struct with values from all sources

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Why mapstructure Tags Are Needed

You might wonder: "Why not just bind `DB_HOST` directly to `Config.DBHost`?"

The answer is that Viper uses an **internal key-value store** and supports **multiple configuration sources**:

### 1. Viper stores everything under keys:

```go
viper.BindEnv("db_host", "DB_HOST")  // Key "db_host" → reads from env var "DB_HOST"
viper.SetDefault("db_host", "localhost")  // Key "db_host" → default value
pflag.String("db_host", ...)  // Flag "db_host" → also maps to key "db_host"
```

### 2. All sources use the same key:

- Env var: `DB_HOST` → stored as key `"db_host"`
- Config file: `db_host: localhost` → stored as key `"db_host"`
- Flag: `--db_host=localhost` → stored as key `"db_host"`
- Default: `SetDefault("db_host", ...)` → stored as key `"db_host"`

### 3. Unmarshal needs to map keys to struct fields:

```go
viper.Unmarshal(&config)
// Viper looks at struct and sees:
// - Field: DBHost (CamelCase)
// - Tag: mapstructure:"db_host" (snake_case)
// - Viper says: "I have key 'db_host', I'll put it in field DBHost"
```

### 4. Why not use struct field names directly?

- Struct fields are `CamelCase` (Go convention): `DBHost`, `DBPort`
- Config keys are `snake_case` (common convention): `db_host`, `db_port`
- Viper can't automatically convert between them
- `mapstructure` tag provides the mapping: `DBHost` ↔ `"db_host"`

### Alternative (without mapstructure)

You *could* use struct field names directly, but you'd need to:
- Use CamelCase for everything: `DBHost` in struct, `DB_HOST` in env, `--DBHost` in flags
- Or use snake_case for struct fields (breaks Go conventions): `db_host` in struct

The `mapstructure` approach gives you:
- Go conventions (CamelCase struct fields)
- Common config conventions (snake_case keys)
- Support for nested configs (e.g., `database.host` in YAML)
- Multiple sources (env, flags, files) all using same keys

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Usage Examples

```bash
# Via environment variable
export PORT=3000
./your-service

# Via command-line flag
./your-service --port 3000

# Via config file (if you add file reading)
./your-service --config config.yaml
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Create a Service](./how-to-create-a-service.md)
