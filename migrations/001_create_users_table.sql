-- Migration: Create users table
--
-- This is the initial migration that creates the users table.
-- Run this SQL in your MySQL database before starting the application.
--
-- HOW TO RUN:
-- mysql -u root -p db_go_basics < migrations/001_create_users_table.sql
--
-- Or using mysql client:
-- mysql> source /path/to/001_create_users_table.sql

-- Create the database if it doesn't exist
CREATE DATABASE IF NOT EXISTS db_go_basics
    CHARACTER SET utf8mb4           -- Supports full Unicode including emojis
    COLLATE utf8mb4_unicode_ci;     -- Case-insensitive Unicode comparison

USE db_go_basics;

-- Create the users table
CREATE TABLE IF NOT EXISTS users (
    -- Primary key using BIGINT UNSIGNED for large scale
    -- AUTO_INCREMENT automatically generates unique IDs
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,

    -- Email with unique constraint prevents duplicates
    -- VARCHAR(255) is the max length for indexed columns in MySQL with utf8mb4
    email VARCHAR(255) NOT NULL,

    -- Password hash storage
    -- bcrypt hashes are always 60 characters, but we use 255 for flexibility
    -- NEVER store plain-text passwords!
    password_hash VARCHAR(255) NOT NULL,

    -- Timestamps for auditing
    -- created_at: When the record was created
    -- updated_at: When the record was last modified
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    -- Soft delete timestamp
    -- NULL = not deleted, timestamp = when it was deleted
    -- All queries must filter: WHERE deleted_at IS NULL
    deleted_at TIMESTAMP NULL DEFAULT NULL,

    -- Primary key constraint
    PRIMARY KEY (id),

    -- Unique constraint on email (excluding soft-deleted users)
    -- This allows re-registration with an email after account deletion
    UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB                     -- InnoDB supports transactions
  DEFAULT CHARSET=utf8mb4           -- Full Unicode support
  COLLATE=utf8mb4_unicode_ci;       -- Case-insensitive comparison

-- Index for soft-delete queries
-- Most queries filter by deleted_at, so this index helps performance
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- Index for email lookups (login, registration check)
-- The UNIQUE constraint already creates an index, but we make it explicit
-- This index includes deleted_at for filtered lookups
CREATE INDEX idx_users_email_active ON users(email, deleted_at);
