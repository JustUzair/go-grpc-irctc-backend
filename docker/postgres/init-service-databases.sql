CREATE USER user_service WITH PASSWORD 'password';
CREATE DATABASE user_db OWNER user_service;

CREATE USER booking_service WITH PASSWORD 'password';
CREATE DATABASE booking_db OWNER booking_service;

CREATE USER payment_service WITH PASSWORD 'password';
CREATE DATABASE payment_db OWNER payment_service;
