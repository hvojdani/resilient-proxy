// file: http_client.c

#include <stdio.h>
#include <string.h>
#include <curl/curl.h>

size_t writeCallback(void *contents, size_t size, size_t nmemb, void *userp)
{
    size_t total = size * nmemb;
    return total;
}

int main(int argc, char** argv)
{
    CURL *curl;
    CURLcode res;

       if (argc < 2)
    {
        printf("Usage: %s <url>\n", argv[0]);
        return 1;
    }

    char *url = argv[1];

    printf("URL: %s\n", url);


    curl_global_init(CURL_GLOBAL_DEFAULT);

    curl = curl_easy_init();
    if (!curl)
    {
        fprintf(stderr, "curl init failed\n");
        return 1;
    }

    curl_easy_setopt(curl, CURLOPT_URL, url);

    // timeouts
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 30L);

    // follow redirects
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);

    // callback
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writeCallback);

    // optional SSL disable
    int sslVerify = 1;
    if (!sslVerify)
    {
        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
        curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
    }

    // better DNS timeout handling
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);

    res = curl_easy_perform(curl);

    if (res != CURLE_OK)
    {
        fprintf(stderr,
                "curl error (%d): %s\n",
                res,
                curl_easy_strerror(res));

        switch (res)
        {
            case CURLE_OPERATION_TIMEDOUT:
                fprintf(stderr, "Timeout occurred\n");
                break;

            case CURLE_COULDNT_RESOLVE_HOST:
            case CURLE_COULDNT_RESOLVE_PROXY:
                fprintf(stderr, "DNS resolution failed\n");
                break;

            case CURLE_SSL_CONNECT_ERROR:
            case CURLE_PEER_FAILED_VERIFICATION:
            case CURLE_SSL_CERTPROBLEM:
                fprintf(stderr, "SSL handshake/certificate failure\n");
                break;

            case CURLE_COULDNT_CONNECT:
                fprintf(stderr, "TCP connection failed\n");
                break;

            case CURLE_RECV_ERROR:
            case CURLE_SEND_ERROR:
                fprintf(stderr, "Network packet loss/interruption\n");
                break;

            case CURLE_GOT_NOTHING:
                fprintf(stderr, "Server closed connection unexpectedly\n");
                break;

            default:
                fprintf(stderr, "Unhandled curl error\n");
                break;
        }
    }
    else
    {
        long httpCode = 0;
        curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &httpCode);

        printf("HTTP status: %ld\n", httpCode);

        if (httpCode >= 500)
        {
            printf("Server-side error\n");
        }
        else if (httpCode >= 400)
        {
            printf("Client-side error\n");
        }
    }

    curl_easy_cleanup(curl);
    curl_global_cleanup();

    return 0;
}