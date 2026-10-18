from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", extra="ignore")

    app_env: str = "development"
    port: int = 8080
    debug: bool = False

    database_url: str = "postgres://excavate:excavate@localhost:5432/excavate?sslmode=disable"

    session_ttl_hours: int = 168
    session_cookie: str = "excavate_session"

    cors_origin: str = "http://localhost:5173"

    max_search_results: int = 8
    max_extract_pages: int = 5
    max_context_chars: int = 24000

    tavily_api_key: str = ""
    gemini_api_key: str = ""
    gemini_model: str = "gemini-3.5-flash"

    research_mode: str = "auto"

    @property
    def secure_cookies(self) -> bool:
        return self.app_env == "production"

    @property
    def real_providers_enabled(self) -> bool:
        if self.research_mode == "mock":
            return False
        if self.research_mode == "real":
            return True
        return bool(self.tavily_api_key and self.gemini_api_key)


@lru_cache
def get_settings() -> Settings:
    return Settings()
