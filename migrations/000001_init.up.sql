CREATE TABLE IF NOT EXISTS users
	(
			uid serial PRIMARY KEY,
			login character(255) NOT NULL,
			password character(64) NOT NULL
	);

CREATE TABLE IF NOT EXISTS user_balance
	(
		id serial PRIMARY KEY,
		uid integer NOT NULL,
		current numeric(5,2) NOT NULL,
		withdrawn numeric(5,2) NOT NULL,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	);

CREATE TABLE IF NOT EXISTS orders
	(
		id serial PRIMARY KEY,
		"number" character(55) NOT NULL,
		status character(125),
		accrual numeric(5,2),
		date timestamp with time zone NOT NULL DEFAULT now(),
		uid integer NOT NULL DEFAULT 1,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	);

CREATE TABLE IF NOT EXISTS withdrawals
	(
		w_id serial PRIMARY KEY,
		"order" character(255) NOT NULL,
		sum numeric(5,2) NOT NULL,
		processed_at timestamp with time zone NOT NULL DEFAULT now(),
		uid integer NOT NULL,
		FOREIGN KEY (uid) REFERENCES users (uid) ON UPDATE CASCADE ON DELETE CASCADE
	);

CREATE UNIQUE INDEX IF NOT EXISTS login_id ON users (login);

CREATE UNIQUE INDEX IF NOT EXISTS order_id ON orders (number);

CREATE UNIQUE INDEX IF NOT EXISTS order_id ON orders (number);
