import React, { Component } from 'react';
import Room from './Room';

class App extends Component {
    constructor(props) {
        super(props);

        this.state = {
            name: 'tester',
            channel: '#gophers',
        }
    }
    render() {
        return (<div>
            name: <br/>
            <input value={this.state.name} onChange={(e) => this.setState({name: e.target.value })} /> <br />

            channel: <br />
            <input value={this.state.channel} onChange={(e) => this.setState({channel: e.target.value })}/> <br/>

            <Room channel={this.state.channel} name={this.state.name} />
        </div>
        );
    }
}


export default App;
