// This script initializes the MongoDB database and users
// It will run when the MongoDB container starts for the first time

// Create the fish_generator database
db = db.getSiblingDB('fish_generator');

// Create collections
db.createCollection('fish');
db.createCollection('news');
db.createCollection('weather');
db.createCollection('prices');
db.createCollection('state');

// Create a specific user for the fish_generator database if needed
// Note: We're already using the root user (fishuser) with the connection string
// This is just for additional database-specific users if required
/*
db.createUser({
  user: "appuser",
  pwd: "apppassword",
  roles: [
    { role: "readWrite", db: "fish_generator" }
  ]
});
*/

// Print completion message
print('MongoDB initialization completed successfully.'); 