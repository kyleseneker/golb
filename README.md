# golb

`golb` is a simple load balancer written in Go. It distributes incoming HTTP requests among multiple backend servers using a round-robin algorithm. The load balancer periodically checks the health of backend servers and routes requests only to healthy servers.

This project was originally created as part of a coding challenge from [codingchallenges.fyi](https://codingchallenges.fyi/challenges/challenge-load-balancer).

## Features

* Round-robin request distribution among backend servers
* Health checking of backend servers
* Graceful handling of backend failures
* Simple configuration via JSON file

## Usage

### Configuration

The load balancer is configured using a JSON file. An example configuration (`config.json`) is provided below:

```json
{
  "health_check_interval": "5s",
  "frontend_port": "9090",
  "backend_urls": ["http://localhost:8080", "http://localhost:8081"]
}
```

* `health_check_interval`: Interval at which the load balancer checks the health of backend servers. Specify in Go duration format (e.g., `"5s"` for 5 seconds).
* `frontend_port`: Port on which the load balancer listens for incoming requests.
* `backend_urls`: List of URLs of backend servers to which the load balancer forwards requests.

### Running the Load Balancer

1. Ensure you have Go installed on your system.

1. Clone the repository:

    ```sh
    git clone https://github.com/kyleseneker/golb.git
    ```

1. Navigate to the project directory:

    ```sh
    cd golb
    ```

1. Modify the `config.json` file to match your configuration.

1. Run the load balancer using the following command:

    ```bash
    go run main.go
    ```

    Alternatively, you can build the binary and run it directly:

    ```bash
    go build -o golb main.go
    ./loadbalancer
    ```

The load balancer will start listening for incoming requests on the specified port.

### Testing

To test the load balancer, you can send HTTP requests to the frontend port using tools like cURL or web browsers. The load balancer will distribute the requests among the configured backend servers.

```bash
curl http://localhost:9090
```

## Contributing

Contributions are welcome! If you find a bug or want to add a new feature, please open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
