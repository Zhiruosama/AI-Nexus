CREATE DATABASE ai_nexus;

USE ai_nexus;

CREATE TABLE test (
  id INT NOT NULL AUTO_INCREMENT,
  message VARCHAR(255),
  PRIMARY KEY (id)
)

INSERT INTO test (message)
VALUES ('helloworld'), ('helloworld2');