import React , {Component} from 'react';
import { graphql, compose } from 'react-apollo';
import gql from 'graphql-tag';

class Room extends Component {
    constructor(props) {
        super(props)

        this.state = {text: ''}
    }

    componentWillMount() {
        this.props.data.subscribeToMore({
            document: Subscription,
            variables: {
                channel: this.props.channel,
            },
            updateQuery: (prev, {subscriptionData}) => {
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
            }
        });
    }

    render() {
        const data = this.props.data;

        if (data.loading) {
            return <div>loading</div>
        }

        return <div>
            <div>
                {data.room.messages.map((msg) =>
                    <div key={msg.id}>{msg.createdBy}: {msg.text}</div>
                )}
            </div>
            <input value={this.state.text} onChange={(e) => this.setState({text: e.target.value})}/>
            <button onClick={() => this.props.mutate({
                variables: {
                    text: this.state.text,
                    channel: this.props.channel,
                    name: this.props.name,
                }
            })} >send</button>
        </div>;
    }
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


export default compose(graphql(Mutation), graphql(Query))(Room);
