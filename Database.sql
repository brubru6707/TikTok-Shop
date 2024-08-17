-- Select the database to use
USE golang_webapp;

-- Create the messages table if it doesn't already exist
CREATE TABLE IF NOT EXISTS messages (
    id INT AUTO_INCREMENT PRIMARY KEY,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

USE golang_webapp;
-- Create the favorites table
CREATE TABLE favorites (
    id INT AUTO_INCREMENT PRIMARY KEY,  -- Primary key for the favorites table
    message_id INT UNIQUE,                     -- Foreign key column referencing messages
    FOREIGN KEY (message_id) REFERENCES messages(id)   -- Foreign key constraint
);
