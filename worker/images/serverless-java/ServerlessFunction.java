import org.json.JSONObject;

import java.lang.reflect.Method;
import java.util.Scanner;

class ServerlessFunction {
    public static void main(String[] args) {
        SendData("started", "");

        Scanner input = new Scanner(System.in);
        Class<?> functionClass = getFunction(input);
        JSONObject success = new JSONObject();
        success.put("type", "function_loaded");
        success.put("data", true);
        System.out.println(success.toString());

        while (true) {
            String line = input.nextLine();
            try {
                JSONObject request = new JSONObject(line);
                if (!request.getString("type").equals("request")) {
                    continue;
                }
                System.out.println("Handling: " + request.toString());
                JSONObject argument = request.getJSONObject("data");
                Object functionInstance = functionClass.getConstructor().newInstance();
                Method functionMethod = functionClass.getMethod("handle", JSONObject.class);
                JSONObject result = (JSONObject)functionMethod.invoke(functionInstance, argument);
                JSONObject response = new JSONObject();
                response.put("type", "response");
                response.put("data", result);
                System.out.println(response.toString());
            } catch(Exception e) {
                System.out.println(e);
                continue;
            }
        }

    }

    private static Class<?> getFunction(Scanner input) {
        while (true){
            String line = input.nextLine();
            try {
                JSONObject obj = new JSONObject(line);

                String type = obj.getString("type");
                if (!type.equals("function")) {
                    continue;
                }
                String function = obj.getString("data");
                StringClassLoader loader = new StringClassLoader(function);
                return loader.findClass("Function");
            } catch(Exception e) {
                System.out.println("failed to load function");
                System.out.println(e);
                continue;
            }
        }
    }

    public static void SendData(String type, String data) {
        JSONObject out = new JSONObject();
        out.put("type", type);
        out.put("data", data);
        System.out.println(out.toString());
    }
}
