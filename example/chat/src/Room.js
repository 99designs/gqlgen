import React, { useState, useEffect, useRef } from 'react';
import styled from 'styled-components';
import gql from 'graphql-tag';
import { useQuery, useMutation } from '@apollo/react-hooks';

export const Chat = styled.div`
    padding: 4px;
    margin: 0 0 12px;
    max-width: 400px;
    max-height: 400px;
    border: 1px solid #ccc;
    overflow-x: hidden;
    overflow-y: scroll;
`;

export const Room = ({ channel, name }) => {
    const messagesEndRef = useRef(null)
    const [ text, setText ] = useState('');

    const [ addMessage ] = useMutation(Mutation, {
        onCompleted: () => {
            setText('');
        }
    });

    const { loading, error, data, subscribeToMore } = useQuery(Query, {
        variables: {
            channel
        },
    });

    // subscribe to more messages
    useEffect(() => {
        const subscription = subscribeToMore({
            document: Subscription,
            variables: {
                channel,
            },
            updateQuery: (prev, { subscriptionData }) => {

                if (!subscriptionData.data) {
                    return prev;
                }
                const newMessage = subscriptionData.data.messageAdded;

                if (prev.room.messages.find((msg) => msg.id === newMessage.id)) {
                    return prev
                }

                return Object.assign({}, prev, {
                    room: Object.assign({}, prev.room, {
                        messages: [...prev.room.messages, newMessage],
                    })
                });
            },
        });


        return () => subscription();

    }, [subscribeToMore]);

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
            {data.room.messages.map((msg) =>
                <div key={msg.id}>
                    {msg.createdBy}: {msg.text}
                </div>
            )}
            <div ref={messagesEndRef} />
        </Chat>

        <input autoFocus value={text} onChange={(e) => setText(e.target.value)} />

        <p>
            <button onClick={() => addMessage({
            variables: {
                text: text,
                channel: channel,
                name: name,
            }
            })} >
                send
            </button>
        </p>
    </>);

}

const Subscription = gql`
    subscription MoreMessages($channel: String!) {
        messageAdded(roomName:$channel) {
            id
            text
            createdBy
        }
    }
`;

const Query = gql`
    query Room($channel: String!) {
        room(name: $channel) {
            messages { id text createdBy }
        }
    }
`;

const Mutation = gql`
    mutation sendMessage($text: String!, $channel: String!, $name: String!) {
        post(text:$text, roomName:$channel, username:$name) { id }
    }
`;
