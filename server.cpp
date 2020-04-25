#include "server.h"
#include <QNetworkDatagram>
#include <QtEndian>

void Server::initSocket()
{
    udpSocket = new QUdpSocket(this);
    udpSocket->bind(QHostAddress::Any, 45467);

    connect(udpSocket, &QUdpSocket::readyRead,
            this, &Server::readPendingDatagrams);
}

void Server::checkIfExists(uint8_t playerNumber, uint32_t count)
{
    if (!inputs[playerNumber].contains(count)) //They are asking for a value we don't have
    {
        if (!buttons[playerNumber].isEmpty())
            inputs[playerNumber][count] = buttons[playerNumber].takeFirst();
        else if (inputs[playerNumber].contains(count-1))
            inputs[playerNumber][count] = inputs[playerNumber][count-1];
        else
            inputs[playerNumber][count] = qMakePair(0, 0/*Controller not present*/);
    }
}

void Server::sendInput(uint32_t count, QHostAddress address, int port, uint8_t playerNum, uint8_t spectator)
{
    char buffer[512];
    buffer[0] = 1; // Key info from server
    buffer[1] = playerNum;
    buffer[2] = 4; //count number
    uint32_t curr = 3;
    for (uint8_t i = 0; i < buffer[2]; ++i)
    {
        if (spectator == 0 || inputs[playerNum].contains(count))
        {
            qToBigEndian(count, &buffer[curr]);
            curr += 4;
            checkIfExists(playerNum, count);
            qToBigEndian(inputs[playerNum][count].first, &buffer[curr]);
            curr += 4;
            buffer[curr] = inputs[playerNum][count].second;
            curr += 1;
        }
        ++count;
    }

    if (curr > 3)
        udpSocket->writeDatagram(&buffer[0], curr, address, port);
}

void Server::readPendingDatagrams()
{
    uint32_t keys, count;
    uint8_t playerNum;
    while (udpSocket->hasPendingDatagrams())
    {
        QNetworkDatagram datagram = udpSocket->receiveDatagram();
        QByteArray incomingData = datagram.data();
        playerNum = incomingData.at(1);
        switch (incomingData.at(0))
        {
            case 0: // key info from client
                count = qFromBigEndian<uint32_t>(&incomingData.data()[2]);
                keys = qFromBigEndian<uint32_t>(&incomingData.data()[6]);
                buttons[playerNum].append(qMakePair(keys, incomingData.at(10)));
                break;
            case 2: // request for player input data
                count = qFromBigEndian<uint32_t>(&incomingData.data()[2]);
                sendInput(count, datagram.senderAddress(), datagram.senderPort(), playerNum, incomingData.at(6));
                break;
            default:
                printf("Unknown packet type %d\n", incomingData.at(0));
                break;
        }
    }
}
