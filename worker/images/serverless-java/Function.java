import org.json.JSONObject;

public class Function{
  public JSONObject handle(JSONObject args){
    System.out.println(args.toString());
    return new JSONObject().put("yolo", "swag");
  }
}
