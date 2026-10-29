-- 0007: credencial do candidato. Hash scrypt no formato
-- scrypt$n$r$p$sal_b64$hash_b64, gerado e conferido pela aplicação.
ALTER TABLE candidato ADD COLUMN senha_hash text;
ALTER TABLE candidato ADD CONSTRAINT candidato_email_unico UNIQUE (email);
