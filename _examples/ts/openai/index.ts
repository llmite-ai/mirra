import OpenAI from 'openai';

async function main() {
  // Check if API key is set
  if (!process.env.OPENAI_API_KEY) {
    console.error('Error: OPENAI_API_KEY environment variable is not set');
    process.exit(1);
  }

  // Initialize OpenAI client with mirra proxy
  const client = new OpenAI({
    apiKey: process.env.OPENAI_API_KEY,
    baseURL: 'http://localhost:4567/v1', // This is where we're configuring mirra
  });

  try {
    console.log('Making request to OpenAI through mirra proxy...\n');

    // Make a simple API call
    const response = await client.chat.completions.create({
      model: 'gpt-4',
      messages: [
        {
          role: 'user',
          content: 'Say hello and tell me a joke',
        },
      ],
      max_tokens: 150,
    });

    // Print the response
    console.log('Response from OpenAI:');
    console.log('---');
    console.log(response.choices[0].message.content);
    console.log('---\n');
    console.log(`Model: ${response.model}`);
    console.log(`Tokens used: ${response.usage?.total_tokens}`);
  } catch (error) {
    if (error instanceof Error) {
      console.error('Error making API call:', error.message);
    } else {
      console.error('Error making API call:', error);
    }
    process.exit(1);
  }
}

main();
