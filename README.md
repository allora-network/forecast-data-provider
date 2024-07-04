Allora Chain Data Pump
======================

Overview
--------
This project serves as a data pump for the Allora Chain, a blockchain built on the Cosmos network.
It restores DB from SQL file and continue to extract and store blockchain data into a PostgreSQL database.

Prerequisites
-------------

*   Go (version 1.21 or later) for backend development.
*   PostgreSQL (version 10 or later) for database management.
*   The `allorad` command-line application for interacting with the Allora Chain.

Setup and Configuration
-----------------------

Ensure the `allorad` application is configured correctly to connect to the Allora Chain node. The PostgreSQL database should be set up with the necessary tables and permissions for data storage. Modify the Go application's configuration to point to the correct database and node endpoints.

Docker Commands
-----------------------
* WORKERS_NUM - Number of workers to process blocks concurrently (default is 5)
* NODE - Allora chain RPC address 
* CONNECTION - Database connection url to store data
* AWS_ACCESS_KEY - AWS access key
* AWS_SECURITY_KEY - AWS security key
* S3_BUCKET_NAME - AWS s3 bucket name
* S3_FILE_KEY - AWS s3 file(SQL backup file) key
* RESET_DB - Flag for restarting database