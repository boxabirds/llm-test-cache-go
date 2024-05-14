# LLM Test Cache

## Problem

Calling Large Language Model (LLM) APIs, such as OpenAI's GPT-3, can quickly become expensive. The cost can escalate to around $60 per million tokens. Additionally, these APIs are challenging to mock effectively, making it difficult to test applications that rely on them without incurring significant costs.

## Objective

This project aims to create a testability experiment that reduces the expensive OpenAI API costs during test time by caching duplicated responses locally. The caching mechanism stores responses based on the `ChatCompletionRequest` parameters, allowing for repeated requests to be served from the cache instead of making costly API calls.

## Approach

The caching mechanism relies on the deterministic results from the APIs, which should occur if the (random) seed parameter is set. If the seed is not set, caching becomes ineffective as a different seed is generated for each request, leading to different responses.

### Key Points:
- **Deterministic Results**: The API should return the same response for the same request if the seed parameter is set.
- **Cache Storage**: Responses are cached locally based on the `ChatCompletionRequest` parameters.
- **Cost Reduction**: By serving repeated requests from the cache, the number of API calls is reduced, leading to significant cost savings.

## Command Line Parameters

When running tests, several command line parameters can be used to control the caching behavior and other settings:

- `-cache-requests`: Enable caching of requests. Default is `false`.
- `-cache-size-limit`: Set the cache size limit in bytes. Default is `10MB` (10 * 1024 * 1024 bytes).
- `-max-tokens`: Set the maximum tokens for the `ChatCompletionRequest`.
- `-keep-cache`: Keep the cache after tests for manual inspection. Default is `false`.
- `-test-cacheability`: Test if the API configuration is deterministic. Default is `false`.

### Example Usage

To run the tests with caching enabled and a cache size limit of 20MB:
`sh go test -v -args -cache-requests -cache-size-limit=20971520`


To run the tests with a maximum token limit of 100 and keep the cache after tests:
`sh go test -v -args -max-tokens=100 -keep-cache`


To run the tests with cacheability testing enabled:
`sh go test -v -args -test-cacheability -max-tokens=100`


### When to Use These Parameters

- **`-cache-requests`**: Use this parameter to enable caching of requests. This is useful when you want to reduce the number of API calls and save costs during testing.
- **`-cache-size-limit`**: Use this parameter to set a limit on the cache size. This helps in managing the disk space used by the cache.
- **`-max-tokens`**: Use this parameter to set the maximum number of tokens for the `ChatCompletionRequest`. This can be useful for testing different token limits.
- **`-keep-cache`**: Use this parameter to keep the cache after tests. This is useful for manual inspection of the cache contents.
- **`-test-cacheability`**: Use this parameter to test if the API configuration is deterministic. This helps in verifying that the API returns consistent responses for the same requests when the seed parameter is set.

By using these parameters, you can effectively manage the caching behavior and control the costs associated with calling LLM APIs during testing.