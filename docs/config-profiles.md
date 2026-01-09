# Config Profiles

bebo supports layered config profiles (base + env + secrets) with validation.

## Example profile
```go
profile := config.Profile{
    BasePath:    "config/base.json",
    EnvPath:     "config/development.json",
    SecretsPath: "config/secrets.json",
    EnvPrefix:   "BEBO_",
    AllowMissing: true,
}

cfg, err := config.LoadProfile(profile)
if err != nil {
    log.Fatal(err)
}

app := bebo.New(bebo.WithConfig(cfg))
```

## Notes
- `AllowMissing` lets missing files (like secrets) be ignored locally.
- Environment variables override the merged JSON files.
- Use `config.Loader[T]` for custom typed config structs.
