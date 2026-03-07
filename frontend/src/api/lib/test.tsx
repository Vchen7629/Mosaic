import { FetchTranscript } from "../hooks/transcript"


const TestButton = () => {
    function testing() {
       const test = FetchTranscript()

       console.log(test)
    }

    return (
        <button 
            onClick={() => testing()}
            className="px-4 py-2 bg-teal-600 hover:bg-teal-700 rounded-md">    
            hi
        </button>
    )
    
}

export default TestButton;