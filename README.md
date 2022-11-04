# delete-telegram
## Disclaimer
I only did this project as a fun afternoon task. You shouldn't depend on it in any circumstance. Always test before using it 
in real-life scenario.


## Usage
- Create a API_ID and API_HASH [here](https://core.telegram.org/api/obtaining_api_id).
- Change configuration in `config.go`'s `defaultConfig`.
- Run application in a **foreign server** (double check connectivity to telegram servers).

### Login (Get verification code):
```shell
curl --location --request POST '{server_ip}:1455/get_code' \
--header 'api-key: secret' \
--header 'Content-Type: application/json' \
--data-raw '{
    "phone": "+989123456789"
}'
```

### Login (Send Verification code [and 2FA if is set]):
```shell
curl --location --request POST '{server_ip}:1455/verify_code' \
--header 'api-key: secret' \
--header 'Content-Type: application/json' \
--data-raw '{
    "code": "90288",
    "user_key": "[from previous API call]",
    "password": ""
}'
```
**SAVE `user_key` in a safe place as it is the only piece of information required to delete your account :)**

### Delete Account:
```shell
curl --location --request POST '{server_ip}:1455/delete' \
--header 'api-key: secret' \
--header 'Content-Type: application/json' \
--data-raw '{
    "user_key": "9ece5aeb-7bbf-4a86-af63-0c583c2a7d8d"
}'
```

## Notes
- **DO NOT** change or modify `data` directory or `DatabaseEncryptionKey` after running the application.
- Users' session data (which is used to connect and access Telegram) are stored in `data` folder. keep it safe. 
- Do not use a single instance (`api_id` and `api_hash`) more than 10 users.
