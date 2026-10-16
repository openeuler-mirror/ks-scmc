#include <string>
#include <iostream>
#include "libsubscribe.h"

using namespace std;

int main()
{
    char addr[] = "127.0.0.1:10050";
    char name[] = "sysadm";
    char password[] = "12345678";
    char tls[] = "false";
    Subscribe(addr, name, password, tls);
    for (;;)
    {
        WarnLog log = GetMessage();
        std::cout << log.id << endl;
    }
}