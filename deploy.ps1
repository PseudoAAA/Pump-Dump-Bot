echo "--- Отправка файлов на VPS ---"
scp -r ./* root@155.212.130.223:/root/PumpDumpBot/
echo "--- Пересборка и запуск на сервере ---"
ssh root@155.212.130.223 "cd /root/PumpDumpBot && docker-compose up -d --build pump-bot && docker-compose logs --tail 20 pump-bot"