CREATE TABLE IF NOT EXISTS api_keys (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    key_hash CHAR(64) NOT NULL,
    owner_name VARCHAR(255) NULL,
    owner_email VARCHAR(255) NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    expires_at DATETIME NULL,
    allowed_ips TEXT NULL,
    allowed_routes TEXT NULL,
    rate_limit_per_minute INT NOT NULL DEFAULT 120,
    last_used_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_api_keys_key_hash (key_hash),
    KEY idx_api_keys_key_prefix (key_prefix),
    KEY idx_api_keys_is_active_expires_at (is_active, expires_at),
    KEY idx_api_keys_owner_email (owner_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS api_key_usage_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    api_key_id BIGINT UNSIGNED NULL,
    key_prefix VARCHAR(32) NULL,
    request_method VARCHAR(16) NOT NULL,
    request_path VARCHAR(512) NOT NULL,
    client_ip VARCHAR(64) NOT NULL,
    user_agent VARCHAR(512) NULL,
    status VARCHAR(32) NOT NULL,
    message VARCHAR(255) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_api_key_usage_logs_api_key_id_created_at (api_key_id, created_at),
    KEY idx_api_key_usage_logs_status_created_at (status, created_at),
    CONSTRAINT fk_api_key_usage_logs_api_key_id
        FOREIGN KEY (api_key_id) REFERENCES api_keys(id)
        ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
