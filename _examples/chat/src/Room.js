import React, { useState, useEffect, useRef } from 'react';
import gql from 'graphql-tag';
import { useQuery, useMutation } from '@apollo/client';
import { Chat, ChatContainer, Message, MessageReceived } from './components/room';

let queuedMessages = []
export const Room = ({ channel, name }) => {
    const messagesEndRef = useRef(null)
    const [ text, setText ] = useState('');

    const [ addMessage ] = useMutation(MUTATION, {
        onCompleted: () => {
            setText('');
        }
    });

    let { loading, error, data, subscribeToMore } = useQuery(QUERY, {
        variables: {
            channel
        },
    });

    if (data && data.room) {
        data = Object.assign({}, data, {
            room: Object.assign({}, data.room, {
                messages: [
                    ...data.room.messages,
                    ...queuedMessages.filter((queuedMessage) => (
                        !data.room.messages.find((msg) => msg.id === queuedMessage.id)
                    )),
                ],
            })
        });
    }

    // subscribe to more messages
    useEffect(() => {
        const subscription = subscribeToMore({
            document: SUBSCRIPTION,
            variables: {
                channel,
            },
            updateQuery: (prev, { subscriptionData }) => {

                if (!subscriptionData.data) {
                    return prev;
                }
                const newMessage = subscriptionData.data.messageAdded;
                if (!prev.room) {
                    queuedMessages.push(newMessage)
                    return prev;
                }

                if (prev.room.messages.find((msg) => msg.id === newMessage.id)) {
                    return prev
                }

                prev = Object.assign({}, prev, {
                    room: Object.assign({}, prev.room, {
                        messages: [
                            ...prev.room.messages,
                            ...queuedMessages.filter((queuedMessage) => (
                                newMessage.id !== queuedMessage.id
                                && !prev.room.messages.find((msg) => msg.id === queuedMessage.id)
                            )),
                            newMessage,
                        ],
                    }),
                });
                queuedMessages = [];
                return prev;
            },
        });


        return () => subscription();

    }, [subscribeToMore, channel]);

    // auto scroll down
    useEffect(() => {
        messagesEndRef && messagesEndRef.current && messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }, [messagesEndRef, data]);

    if (loading) {
        return <div>loading</div>
    }

    if (error) {
        return <div>error</div>
    }

    return (<>
        <Chat>
            <ChatContainer>
                {data.room.messages.map((msg) =>
                    msg.createdBy === name ? <Message key={msg.id}>
                        {msg.text}
                    </Message> : <MessageReceived key={msg.id}>
                        <span>{msg.createdBy}</span>
                        {msg.text}
                    </MessageReceived>
                )}
            </ChatContainer>
            <div ref={messagesEndRef} />
        </Chat>

        <input value={text} onChange={(e) => setText(e.target.value)} />

        <p>
            <button
                onClick={() => addMessage({
                    variables: {
                        text: text,
                        channel: channel,
                        name: name,
                    }
                })}>
                send
            </button>
        </p>
    </>);

}

const SUBSCRIPTION = gql`
    subscription MoreMessages($channel: String!) {
        messageAdded(roomName:$channel) {
            id
            text
            createdBy
        }
    }
`;

const QUERY = gql`
    query Room($channel: String!) {
        room(name: $channel) {
            messages { id text createdBy }
        }
    }
`;

const MUTATION = gql`
    mutation sendMessage($text: String!, $channel: String!, $name: String!) {
        post(text:$text, roomName:$channel, username:$name) { id }
    }
`;
