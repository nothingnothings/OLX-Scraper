<?php

use PhpAmqpLib\Connection\AMQPStreamConnection;
use PhpAmqpLib\Message\AMQPMessage;

require __DIR__ . "/vendor/autoload.php";

class SpreedSheet
{
    public $service;

    private array $existingIds = [];

    function __construct()
    {
        $client = new \Google_Client();
        $client->setApplicationName("Google Sheets and PHP");
        $client->setScopes([\Google_Service_Sheets::SPREADSHEETS]);
        $client->setAccessType('offline');
        $client->setAuthConfig(__DIR__ . "/config/" . getenv("SPREED_SHEET_AUTH"));
        $this->service = new Google_Service_Sheets($client);
    }

    function loadExistingIds()
    {
        $response = $this->service->spreadsheets_values->get(
            getenv("SPREED_SHEET_ID"),
            "olx!A:A"
        );

        $values = $response->getValues();

        foreach ($values as $row) {
            if (isset($row[0])) {
                $this->existingIds[] = (string)$row[0];
            }
        }

        print_r("Loaded " . count($this->existingIds) . " existing IDs\n");
    }


    private function insert($item)
    {
        try {
            $range = "olx!A2:I2";
    
            $values = [[
                $item["id"],
                $item["title"],
                $item["created_at"],
                $item["url"],
                $item["location"],
                $item["image"],
                $item["price"],
                $item["parameters"],
                $item["description"]
            ]];
    
            $body = new Google_Service_Sheets_ValueRange([
                "values" => $values
            ]);
    
            $result = $this->service->spreadsheets_values->append(
                getenv("SPREED_SHEET_ID"),
                $range,
                $body,
                [
                    "valueInputOption" => "USER_ENTERED",
                    "insertDataOption" => "INSERT_ROWS"
                ]
            );
    
        } catch (Exception $e) {
            echo "Google Sheets error:\n";
            echo $e->getMessage() . PHP_EOL;
        }
    }


    function isDataExist($searchValue, $searchSheet = null, $searchRange = null)
    {
        return in_array(
            (string)$searchValue,
            $this->existingIds
        );
    }

    function sendData($data, $target)
    {
        $connection = new AMQPStreamConnection(getenv("RABBITMQ_SERVER"), getenv("RABBITMQ_PORT"), getenv("RABBITMQ_DEFAULT_USER"), getenv("RABBITMQ_DEFAULT_PASS"), getenv("RABBITMQ_DEFAULT_VHOST"));
        $channel = $connection->channel();
        if (!$connection->isConnected()) {
            throw new Exception('Connection RabbitMQ Failed. \n');
        } else {
            print_r("Connection RabbitMQ Success \n");
        }
        $channel->queue_declare(getenv("RABBITMQ_DEFAULT_QUEUE"), false, false, false, false);

        foreach ($data as $item) {
                $this->insert($item);
                $message = "*{$item["title"]}*\n\n"
                    . "*Preço:* {$item["price"]}\n\n"
                    . "*ID:* {$item["id"]}\n\n"
                    . "*Criado em:*\n{$item["created_at"]}\n\n"
                    . "*Localização:*\n _{$item["location"]}_\n\n"
                    . "*Imagem:*\n{$item["image"]}\n\n"
                    . "*Detalhes:* \n{$item["parameters"]}\n\n"
                    . "*Link:*\n _{$item["url"]}_\n\n"
                    . "\n\n"
                    . "*Usuário* \n"
                    . "\n\n"
                    . "*Nome:* {$item["user"]}\n\n"
                    . "*Avaliação:*\n {$item["rating"]}\n\n";
                $message = [
                    "Target" => $target,
                    "Message" => $message,
                    "Image" => $item["thumbnail"]
                ];
                $msg = new AMQPMessage(json_encode($message));
                $channel->basic_publish($msg, '', 'wa-text');
                print_r("send message id :" . $item['id'] . "\n");
        }

        $channel->close();
        $connection->close();
        print_r("Connection RabbitMQ Closed \n");
    }
}
