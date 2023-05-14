# Chat Proxy

Chat Proxy is a project that enables users to utilize the power of OpenAI's GPT-4 model for generating responses based on their input queries. It provides a simple interface for interacting with the model and storing the chat history along with an audit trail.

## Features

- Setup chat-purpose to give context to the conversation
- Send and receive chat messages
- Maintain conversation history
- Respond to user queries based on the GPT-4 completion
- Load configuration files and messages from filesystem
- Write an audit trail to a text file

## Dependencies

The project requires the following dependencies:

- Go 1.17 or newer
- github.com/sashabaranov/go-openai (GPT-4 client library)
- github.com/google/go-cmp (for test comparison)

## Installation

> Make sure you have Go 1.17 or newer installed on your system.

```sh
git clone https://github.com/<your_username>/chatproxy.git
cd chatproxy
go build
```

## Usage

1. Set up environment variable `OPENAPI_TOKEN` with your OpenAI API token:

   ```sh
   export OPENAPI_TOKEN=<your_openai_api_token>
   ```

2. Run the binary:

   ```sh
   ./chatproxy
   ```

3. Follow the command line instructions to start your conversation.

## File Upload
Chat Proxy supports uploading files to include their content as part of the conversation context. This feature allows users to provide additional information to the AI model by uploading text from files.

To include files in your conversation:

1. Place your text files in a folder that is accessible by the Chat Proxy program. The files should have a `.txt` extension and should not be hidden files (i.e., having filenames starting with a `.` character).

2. During your conversation, prepend a `>` character to the file or folder path to upload its content as a message.

```sh
>path/to/folder_or_file
```

Example:

Assuming you have a folder called `context_files` with multiple text files, you can simply upload these files by entering this line when prompted for a message:

```sh
>context_files
```

The contents of the text files will then be added as messages to the chat context with a user role, and GPT-4 will be able to consider them when responding to your queries.

**Note:** You can use relative or absolute paths for files and folders. assistant}



## Contributing

We welcome contributions to this project. If you'd like to contribute, please open an issue or submit a pull request.

## License

This project is licensed under the [MIT License](LICENSE.md).