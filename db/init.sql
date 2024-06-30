CREATE DATABASE compman_db;
\c compman_db;
CREATE TABLE IF NOT EXISTS companies (
                           id UUID PRIMARY KEY,
                           name VARCHAR(15) NOT NULL UNIQUE,
                           description VARCHAR(3000),
                           employee_count INT NOT NULL,
                           registered BOOLEAN NOT NULL,
                           type INT NOT NULL
);

