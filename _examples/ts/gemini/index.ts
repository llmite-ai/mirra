import { GoogleGenerativeAI } from "@google/generative-ai";

async function main() {
  const apiKey = process.env.GEMINI_API_KEY;
  if (!apiKey) {
    console.error("Error: GEMINI_API_KEY environment variable not set");
    process.exit(1);
  }

  try {
    const genAI = new GoogleGenerativeAI(apiKey);

    // Configure to use mirra proxy
    const model = genAI.getGenerativeModel(
      { model: "gemini-2.0-flash-exp" },
      { baseUrl: "http://localhost:4567" } // This is where we're configuring mirra
    );

    const prompt = "Say hello and a joke";
    const result = await model.generateContent(prompt);
    const response = result.response;
    const text = response.text();

    if (!text) {
      console.log("No response from Gemini.");
      return;
    }

    console.log(text);
  } catch (error) {
    console.error("Error:", error);
    process.exit(1);
  }
}

main();
